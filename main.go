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
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hacbs-release/release-availability-metrics/pkg/checks"

	"github.com/containers/storage/pkg/reexec"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	pollInterval int

	quay   *checks.QuayCheck
	github *checks.GitCheck
	gitlab *checks.GitCheck

	logger = log.New(os.Stdout, "metrics-server: ", log.LstdFlags)
)

func collectAndRecord(ctx context.Context) {
	// service checks
	serviceChecks := strings.Split(os.Getenv("SERVICE_CHECKS"), ",")

	// default internal
	pollInterval, _ = strconv.Atoi(os.Getenv("SERVICE_POLL_INTERVAL"))
	if pollInterval == 0 {
		pollInterval = 60
	}

	// load quay.io configs
	quayUsername := os.Getenv("QUAY_USERNAME")
	quayPassword := os.Getenv("QUAY_PASSWORD")
	quayImagePullSpec := os.Getenv("QUAY_IMAGE_PULLSPEC")
	quayImageTags := os.Getenv("QUAY_IMAGE_TAGS")

	// load github configs
	githubRepoUrl := os.Getenv("GITHUB_REPO_URL")
	githubRevision := os.Getenv("GITHUB_REVISION")
	githubPath := os.Getenv("GITHUB_PATH")
	githubToken := os.Getenv("GITHUB_TOKEN")

	// load gitlab configs
	gitlabRepoUrl := os.Getenv("GITLAB_REPO_URL")
	gitlabRevision := os.Getenv("GITLAB_REVISION")
	gitlabPath := os.Getenv("GITLAB_PATH")
	gitlabToken := os.Getenv("GITLAB_TOKEN")

	logger.Println(fmt.Sprintf("environment loaded. Poll interval: %d * time.Second", pollInterval))

	go func() {
		for {
			// run checks
			for i := 0; i < len(serviceChecks); i++ {
				switch svc := serviceChecks[i]; svc {
				case "quay":
					if quay == nil {
						auth := checks.NewQuayAuth(quayUsername, quayPassword)
						quay = checks.NewQuayCheck(
							ctx,
							auth,
							quayImagePullSpec,
							strings.Split(quayImageTags, ","),
							logger)
					}
					quay.Check()
				case "github":
					if github == nil {
						github = checks.NewGitCheck(
							"github",
							githubToken,
							githubRepoUrl,
							githubRevision,
							githubPath,
							logger)
					}
					github.Check()
				case "gitlab":
					if gitlab == nil {
						gitlab = checks.NewGitCheck(
							"gitlab",
							gitlabToken,
							gitlabRepoUrl,
							gitlabRevision,
							gitlabPath,
							logger)
					}
					gitlab.Check()
				}
			}
			time.Sleep(time.Duration(pollInterval) * time.Second)
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

	listenPort := os.Getenv("SERVICE_LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8080"
	}
	logger.Println(fmt.Sprintf("server starting at :%s", listenPort))
	err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil)
	if err != nil {
		logger.Println(err.Error())
	}
}
