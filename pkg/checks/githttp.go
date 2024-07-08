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
	"log"
	"net/http"
	"net/url"
	"regexp"

	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

// defines the GitHttpCheck type.
type GitHttpCheck struct {
	name           string
	repositoryType int
	projectId      string
	token          string
	url            string
	revision       string
	path           string
	log            *log.Logger
}

// NewGitHttpCheck returns a new instance of GitHttpCheck.
func NewGitHttpCheck(name string, repositoryType int, projectId string, token string, url string, revision string,
	path string, log *log.Logger) *GitHttpCheck {
	newCheck := &GitHttpCheck{
		name:           name,
		repositoryType: repositoryType,
		projectId:      projectId,
		token:          token,
		url:            url,
		revision:       revision,
		path:           path,
		log:            log,
	}
	return newCheck
}

// fetchRemoteFile connects to a remote git url and return a instance of CheckResult and nil in case of success or a
// instance of CheckResult and error in case of failure.
func (c *GitHttpCheck) fetchRemoteFile() (CheckResult, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	gitUrl := c.getRawGitUrl(c.url, c.revision, c.path)
	req, _ := http.NewRequest("GET", gitUrl, nil)
	if c.repositoryType == GITLAB_REPO {
		req.Header.Set("PRIVATE-TOKEN", c.token)
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
func (c *GitHttpCheck) Check() float64 {
	var reason string

	c.log.Println("running git check to ", c.url)
	res, err := c.fetchRemoteFile()
	if err != nil {
		reason = err.Error()
	}
	metrics.RecordAvailabilityData(c.name, reason, res.status, res.code)

	return res.code
}

// getRawGitUrl rewrites the given url to a URL that can be reached with simple authentication.
func (c *GitHttpCheck) getRawGitUrl(url string, revision string, path string) string {
	var gitUrl string

	if c.repositoryType == GITHUB_REPO {
		re := regexp.MustCompile(`github.com`)
		rawGithubUrl := re.ReplaceAll([]byte(url), []byte("raw.githubusercontent.com"))
		gitUrl = fmt.Sprintf("%s/%s/%s", rawGithubUrl, revision, path)
	} else if c.repositoryType == GITLAB_REPO {
		gitUrl, _ = c.getGitlabFilesUrl()
	}

	return gitUrl
}

// getGitlabFileUrl returns the gitlab files API URL that can be reachead using a gitlab user token.
func (c *GitHttpCheck) getGitlabFilesUrl() (string, error) {
	u, err := url.Parse(c.url)
	if err != nil {
		c.log.Println(err.Error())
		return "", err
	}
	gitlabFilesUrl := fmt.Sprintf("%s://%s/api/v4/projects/%s/repository/files/%s?ref=%s", u.Scheme, u.Host,
		c.projectId, url.PathEscape(c.path), c.revision)

	return gitlabFilesUrl, nil
}
