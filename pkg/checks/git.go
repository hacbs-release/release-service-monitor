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
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/storage/memory"

	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// defines the GitCheck type.
type GitCheck struct {
    prefix   string
	name     string
	token    string
	url      string
	revision string
	path     string
	log      *log.Logger
    metric   metrics.GaugeMetric
}

// NewGitCheck returns a new instance of GitCheck.
func NewGitCheck(prefix string, name string, token string, url string, revision string, path string,
	log *log.Logger, metric metrics.GaugeMetric) *GitCheck {
	newCheck := &GitCheck{
        prefix:   prefix,
		name:     name,
		token:    token,
		url:      url,
		revision: revision,
		path:     path,
		log:      log,
        metric:   metric,
	}
	return newCheck
}

// getInsecureClient return a new instance of http.Client with InsecureSkipVerify set to true.
func (c *GitCheck) getInsecureClient() *http.Client {
	insecureClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 15 * time.Second,
	}
	return insecureClient
}

// cloneAndGetTree clone a git repository and returns a new instance of object.Tree.
func (c *GitCheck) cloneAndGetTree() (*object.Tree, error) {
	var tree *object.Tree

	insecureClient := c.getInsecureClient()
	client.InstallProtocol("https", githttp.NewClient(insecureClient))
	cloneOptions := &git.CloneOptions{
		URL: c.url,
		Auth: &githttp.BasicAuth{
			Username: "oauth2",
			Password: c.token,
		},
		ReferenceName: plumbing.ReferenceName(c.revision),
		Progress:      io.Discard,
		Depth:         1,
	}

	r, err := git.Clone(memory.NewStorage(), nil, cloneOptions)
	if err != nil {
		c.log.Println(err.Error())
		return tree, err
	}

	ref, err := r.Head()
	if err != nil {
		c.log.Println(err.Error())
		return tree, err
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		c.log.Println(err.Error())
		return tree, err
	}

	tree, err = commit.Tree()
	if err != nil {
		c.log.Println(err.Error())
		return tree, err
	}

	return tree, nil
}

// statFile checks the existence of a file in the git repositry.
func (c *GitCheck) statFile() (CheckResult, error) {
	tree, err := c.cloneAndGetTree()
	if err != nil {
		c.log.Println(fmt.Sprintf("%s check failed (%s)", c.name, err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}

	_, err = tree.File(c.path)
	if err != nil {
		c.log.Println(fmt.Sprintf("%s check failed (%s)", c.name, err.Error()))
		return CheckResult{1, "Failed", err.Error()}, err
	}
	c.log.Println(c.name, "check succeeded")

	return CheckResult{0, "Succeeded", ""}, nil
}

// Check runs a check and returns a float64 of the check result. The float64 is required to push values
// to prometheus.
func (c *GitCheck) Check() float64 {
	var reason string

	res, err := c.statFile()
	if err != nil {
		reason = err.Error()
	}
    c.metric.Record([]string{c.name, reason, res.status}, res.code)

	return res.code
}

func (c *GitCheck) GetMetric() (metrics.GaugeMetric) {
    return c.metric
}
