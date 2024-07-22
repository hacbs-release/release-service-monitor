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
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

// defines the HttpCheck type.
type HttpCheck struct {
	name     string
	username string
	password string
	url      string
	cert     string
	key      string
	insecure bool
	follow   bool
	scheme   string
	host     string
	path     string
	log      *log.Logger
	metric   metrics.CompositeMetric
}

// NewHttpCheck returns a new instance of HttpCheck.
func NewHttpCheck(name, username, password, url, cert, key string, insecure, follow bool, log *log.Logger,
	metric metrics.CompositeMetric) *HttpCheck {
	newCheck := &HttpCheck{
		name:     name,
		username: username,
		password: password,
		url:      url,
		cert:     cert,
		key:      key,
		insecure: insecure,
		follow:   follow,
		log:      log,
		metric:   metric,
	}
	newCheck.parseUrl()

	return newCheck
}

// parseUrl parses the given url to the constructor function and adds the url parts to scheme, host and path parameters.
func (c *HttpCheck) parseUrl() {
	re := regexp.MustCompile(`(http.?)://([a-z\-\.]+)(/(.*))?`)
	if re != nil {
		parts := re.FindSubmatch([]byte(c.url))
		c.scheme = string(parts[1])
		c.host = string(parts[2])
		c.path = string(parts[3])
	}
}

// fetchRemoteFile connects to a remote git url and return a instance of CheckResult and nil in case of success or a
// instance of CheckResult and error in case of failure.
func (c *HttpCheck) checkUrl() (CheckResult, error) {
	clientTLSCert, err := tls.X509KeyPair([]byte(c.cert), []byte(c.key))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.insecure,
			Certificates:       []tls.Certificate{clientTLSCert},
		},
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if c.follow == false {
				return http.ErrUseLastResponse
			} else {
				return nil
			}
		},
	}

	req, _ := http.NewRequest("GET", c.url, nil)
	if c.username != "" && c.password != "" {
		data := []byte(fmt.Sprintf("%s:%s", c.username, c.password))
		encodedCredentials := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(encodedCredentials, data)
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encodedCredentials))
	}
	resp, err := client.Do(req)

	if err != nil {
		c.log.Println(fmt.Sprintf("%s check failed (%s)", c.name, err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	if resp.StatusCode != 200 {
		c.log.Println(fmt.Sprintf("%s check failed (%s)", c.name, resp.Status))
		return CheckResult{1, "Failed", resp.Status}, err
	}

	c.log.Println(c.name, "check succeeded")

	return CheckResult{0, "Succeeded", ""}, nil
}

// Check runs a check and returns a float64 of the check result. The float64 is required to push values
// to prometheus.
func (c *HttpCheck) Check() float64 {
	var reason string

	c.log.Println("running HTTP check:", c.name)
	res, err := c.checkUrl()
	if err != nil {
		reason = err.Error()
	}
	c.metric.Gauge.Record([]string{c.name}, metrics.FlipValue(res.code))
	c.metric.Histogram.Record([]string{c.name, reason, res.status}, 1)

	return res.code
}
