package main

import (

    "os"
	"time"
    "strings"
    "context"
	"net/http"

    "github.com/hacbs-release/release-availability-metrics/pkg/checks"

    "github.com/containers/storage/pkg/reexec"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func collectAndRecord(ctx context.Context) {

    // move to a pkg
    quayUsername      := os.Getenv("QUAY_USERNAME")
    quayPassword      := os.Getenv("QUAY_PASSWORD")
    quayImagePullSpec := os.Getenv("QUAY_IMAGE_PULLSPEC")
    quayImageTags     := os.Getenv("QUAY_IMAGE_TAGS")

    auth := checks.NewQuayAuth(quayUsername, quayPassword)

    quay   := checks.NewQuayCheck(ctx, auth, quayImagePullSpec, strings.Split(quayImageTags, ","))
    gitlab := checks.NewGitlabCheck("", "gitlab.com", "main", "file.txt")
    github := checks.NewGithubCheck("", "github.com", "main", "file.txt")

    go func() {
        for {
            // run checks, serialized for the moment
            quay.Check()
            gitlab.Check()
            github.Check()

            time.Sleep(60 * time.Second)
        }
    }()
}

func main() {

    var ctx context.Context

    if reexec.Init() {
	    return
	}

    ctx = context.Background()
	collectAndRecord(ctx)
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":8082", nil)
}
