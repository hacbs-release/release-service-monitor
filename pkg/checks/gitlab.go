package checks

import (
    "github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

type gitlabCheck struct {
    name string
    token string
    url string
    revision string
    path string
}

func NewGitlabCheck(token string, url string, revision string, path string) (*gitlabCheck) {
    newCheck := &gitlabCheck{
        name: "gitlab",
        token: token, // optional for private access
        url: url,
        revision: revision,
        path: path,
    }
    return newCheck
}

func (c *gitlabCheck) Check() (int) {
    metrics.RecordAvailabilityData(c.name, "Not implemented", "Failed", 1)

    return 0
}
