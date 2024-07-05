package checks

import (
    "github.com/hacbs-release/release-availability-metrics/pkg/metrics"
)

type githubCheck struct {
    name string
    token string
    url string
    revision string
    path string
}

func NewGithubCheck(token string, url string, revision string, path string) (*githubCheck) {
    newCheck := &githubCheck{
        name: "github",
        token: token,
        url: url,
        revision: revision,
        path: path,
    }
    return newCheck
}

func (c *githubCheck) Check() (int) {
    metrics.RecordAvailabilityData(c.name, "Not implemented", "Failed", 1)

    return 0
}
