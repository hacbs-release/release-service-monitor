/*
Copyright 2024 Red Hat Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

// QuayCheck sets the necessary parameters to run a check to a container registry.
type QuayCheck struct {
	auth   QuayAuth
	name   string
	image  string
	tags   []string
	log    *log.Logger
	metric metrics.CompositeMetric
	client *http.Client
}

// NewQuayCheck creates a new QuayCheck instance.
func NewQuayCheck(
	auth *QuayAuth,
	name, image string,
	tags []string,
	log *log.Logger,
	metric metrics.CompositeMetric,
) *QuayCheck {
	log.Println("creating new Quay check")
	return &QuayCheck{
		auth:   *auth,
		name:   name,
		image:  image,
		tags:   tags,
		log:    log,
		metric: metric,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// parseImageRef parses an image reference into registry and repository.
// Example: quay.io/konflux-ci/release-service-utils -> registry=quay.io, repo=konflux-ci/release-service-utils
func (c *QuayCheck) parseImageRef() (registry, repo string) {
	parts := strings.SplitN(c.image, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		registry = parts[0]
		repo = parts[1]
	} else {
		// Default to docker.io for images without explicit registry
		registry = "docker.io"
		repo = c.image
	}
	return registry, repo
}

var (
	realmRe   = regexp.MustCompile(`realm="([^"]+)"`)
	serviceRe = regexp.MustCompile(`service="([^"]+)"`)
)

// getAuthToken retrieves a bearer token for the registry using the WWW-Authenticate challenge.
func (c *QuayCheck) getAuthToken(ctx context.Context, registry, repo, wwwAuth string) (string, error) {
	// Parse WWW-Authenticate header
	// Example: Bearer realm="https://quay.io/v2/auth",service="quay.io",scope="repository:user/repo:pull"
	realmMatch := realmRe.FindStringSubmatch(wwwAuth)
	serviceMatch := serviceRe.FindStringSubmatch(wwwAuth)

	if len(realmMatch) < 2 {
		return "", fmt.Errorf("failed to parse auth realm from: %s", wwwAuth)
	}

	realm := realmMatch[1]
	service := ""
	if len(serviceMatch) >= 2 {
		service = serviceMatch[1]
	}

	// Build token request URL
	tokenURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", realm, service, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", tokenURL, nil)
	if err != nil {
		return "", err
	}

	// Add basic auth if credentials provided
	if c.auth.getUsername() != "" && c.auth.getPassword() != "" {
		req.SetBasicAuth(c.auth.getUsername(), c.auth.getPassword())
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("authentication failed (status %d) - check credentials", resp.StatusCode)
		}
		return "", fmt.Errorf("token request failed with status: %d", resp.StatusCode)
	}

	var tokenResp struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	return tokenResp.AccessToken, nil
}

// checkManifest checks if a manifest exists for the given image and tag using the registry API.
func (c *QuayCheck) checkManifest(ctx context.Context, tag string) error {
	registry, repo := c.parseImageRef()
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repo, tag)

	acceptHeader := strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", ")

	resp, err := c.doManifestRequest(ctx, manifestURL, acceptHeader, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If unauthorized, try to get a token and retry
	if resp.StatusCode == http.StatusUnauthorized {
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if wwwAuth == "" {
			return fmt.Errorf("unauthorized and no WWW-Authenticate header")
		}

		token, err := c.getAuthToken(ctx, registry, repo, wwwAuth)
		if err != nil {
			return fmt.Errorf("failed to get auth token: %v", err)
		}

		resp, err = c.doManifestRequest(ctx, manifestURL, acceptHeader, token)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("manifest check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// doManifestRequest performs a HEAD request to the manifest URL with optional auth token.
func (c *QuayCheck) doManifestRequest(ctx context.Context, url, acceptHeader, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", acceptHeader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.client.Do(req)
}

// checkImage verifies that all configured tags are accessible via the registry API.
func (c *QuayCheck) checkImage(ctx context.Context) (CheckResult, error) {
	c.log.Printf("checking manifest for %s\n", c.getImage())

	for _, tag := range c.tags {
		if tag == "" {
			tag = "latest"
		}

		if err := c.checkManifest(ctx, tag); err != nil {
			c.log.Printf("[ERROR] %s:%s check failed: %v\n", c.name, tag, err)
			return CheckResult{1, "Failed", err.Error()}, err
		}
	}

	c.log.Println(c.name, "check succeeded")
	return CheckResult{0, "Succeeded", ""}, nil
}

// Check runs a QuayCheck and returns the float64 status required to save the prometheus data.
func (c *QuayCheck) Check(ctx context.Context) float64 {
	var reason string

	c.log.Println("running quay check:", c.name)
	result, err := c.checkImage(ctx)
	if err != nil {
		reason = err.Error()
	}
	c.metric.Gauge.Record([]string{c.name}, metrics.FlipValue(result.code))
	c.metric.Histogram.Record([]string{c.name, reason, result.status}, 1)

	return result.code
}

// getImage returns the image parameter of a QuayCheck instance.
func (c *QuayCheck) getImage() string {
	return c.image
}
