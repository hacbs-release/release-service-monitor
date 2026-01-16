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
	"fmt"
	"io"
	"log"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	//metric   metrics.GaugeMetric
	metric metrics.CompositeMetric
}

// NewGitCheck returns a new instance of GitCheck.
func NewGitCheck(prefix string, name string, token string, url string, revision string, path string,
	log *log.Logger, metric metrics.CompositeMetric) *GitCheck {
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

// cloneAndGetTree clone a git repository and returns a new instance of object.Tree.
func (c *GitCheck) cloneAndGetTree(ctx context.Context) (*object.Tree, error) {
	var tree *object.Tree

	cloneOptions := &git.CloneOptions{
		URL:           c.url,
		ReferenceName: plumbing.ReferenceName(c.revision),
		Progress:      io.Discard,
		Depth:         1,
	}

	// Only set auth if token is provided
	if c.token != "" {
		cloneOptions.Auth = &githttp.BasicAuth{
			Username: "oauth2",
			Password: c.token,
		}
	}

	r, err := git.CloneContext(ctx, memory.NewStorage(), nil, cloneOptions)
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

// statFile checks the existence of a file in the git repository.
func (c *GitCheck) statFile(ctx context.Context) (CheckResult, error) {
	tree, err := c.cloneAndGetTree(ctx)
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
func (c *GitCheck) Check(ctx context.Context) float64 {
	var reason string

	c.log.Println("running git check:", c.name)
	res, err := c.statFile(ctx)
	if err != nil {
		reason = err.Error()
	}
	c.metric.Gauge.Record([]string{c.name}, metrics.FlipValue(res.code))
	c.metric.Histogram.Record([]string{c.name, reason, res.status}, 1)

	return res.code
}

func (c *GitCheck) GetMetric() metrics.CompositeMetric {
	return c.metric
}
