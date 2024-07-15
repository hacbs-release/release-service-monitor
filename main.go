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
	"strings"
	"time"

	"github.com/hacbs-release/release-availability-metrics/pkg/checks"
	"github.com/hacbs-release/release-availability-metrics/pkg/config"
	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/containers/storage/pkg/reexec"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	pollInterval int

	git   []*checks.GitCheck
	quay  []*checks.QuayCheck
	_http []*checks.HttpCheck

	logger = log.New(os.Stdout, "metrics-server: ", log.LstdFlags)
)

func collectAndRecord(ctx context.Context, cfg *config.Config) {
	// default internal
	pollInterval = cfg.Service.PollInterval
	if pollInterval == 0 {
		pollInterval = 60
	}
	logger.Println(fmt.Sprintf("Poll interval: %d * time.Second", pollInterval))

	// registering metrics
	prefix := cfg.Service.MetricsPrefix
	if prefix == "" {
		prefix = "metrics_server"
	}
	metric := metrics.NewGaugeMetric(cfg.Service.MetricsPrefix, []string{"check", "reason", "status"})
	prometheus.MustRegister(metric.Metric)

	// instance git checks, if defined
	if len(cfg.Checks.Git) != 0 {
		for i := 0; i < len(cfg.Checks.Git); i++ {
			gitCheck := cfg.Checks.Git[i]
			// get the token from env if not specified in config
			token := os.Getenv(fmt.Sprintf("%s_GIT_TOKEN", strings.ToUpper(gitCheck.Name)))
			if token == "" {
				token = gitCheck.Token
			}
			newCheck := checks.NewGitCheck(
				cfg.Service.MetricsPrefix,
				gitCheck.Name,
				token,
				gitCheck.Url,
				gitCheck.Revision,
				gitCheck.Path,
				logger,
				metric)
			git = append(git, newCheck)
		}
	}

	// instance quay checks, if defined
	if len(cfg.Checks.Quay) != 0 {
		for i := 0; i < len(cfg.Checks.Quay); i++ {
			quayCheck := cfg.Checks.Quay[i]
			username := os.Getenv(fmt.Sprintf("%s_QUAY_USERNAME", strings.ToUpper(quayCheck.Name)))
			password := os.Getenv(fmt.Sprintf("%s_QUAY_PASSWORD", strings.ToUpper(quayCheck.Name)))
			if username == "" {
				username = quayCheck.Username
			}
			if password == "" {
				password = quayCheck.Password
			}
			auth := checks.NewQuayAuth(quayCheck.Username, quayCheck.Password)
			newCheck := checks.NewQuayCheck(
				ctx,
				auth,
				quayCheck.Name,
				quayCheck.PullSpec,
				quayCheck.Tags,
				logger,
				metric)
			quay = append(quay, newCheck)
		}
	}

	// instance http checks, if defined
	if len(cfg.Checks.Http) != 0 {
		for i := 0; i < len(cfg.Checks.Http); i++ {
			httpCheck := cfg.Checks.Http[i]
			username := os.Getenv(fmt.Sprintf("%s_HTTP_USERNAME", strings.ToUpper(httpCheck.Name)))
			password := os.Getenv(fmt.Sprintf("%s_HTTP_PASSWORD", strings.ToUpper(httpCheck.Name)))
			if username == "" {
				username = httpCheck.Username
			}
			if password == "" {
				password = httpCheck.Password
			}
			newCheck := checks.NewHttpCheck(
				httpCheck.Name,
				username,
				password,
				httpCheck.Url,
				httpCheck.Follow,
				logger,
				metric)
			_http = append(_http, newCheck)
		}
	}

	go func() {
		for {
			// run git checks, if defined
			if len(git) != 0 {
				for i := 0; i < len(git); i++ {
					git[i].Check()
				}
			}
			// run http checks, if defined
			if len(_http) != 0 {
				for i := 0; i < len(quay); i++ {
					_http[i].Check()
				}
			}
			// run quay checks, if defined
			if len(quay) != 0 {
				for i := 0; i < len(quay); i++ {
					quay[i].Check()
				}
			}
			time.Sleep(time.Duration(pollInterval) * time.Second)
		}
	}()
}

func main() {
	var ctx context.Context

	ctx = context.Background()

	if reexec.Init() {
		return
	}

	cfgFilePath := "server-config.yaml"
	if len(os.Args) > 1 {
		cfgFilePath = os.Args[1]
	}

	logger.Println(fmt.Sprintf("loading config from: %s ", cfgFilePath))
	cfg, err := config.LoadConfig(cfgFilePath)
	if err != nil {
		panic(err)
	}

	collectAndRecord(ctx, &cfg)
	http.Handle("/metrics", promhttp.Handler())

	listenPort := cfg.Service.ListenPort
	if listenPort == 0 {
		listenPort = 8080
	}
	logger.Println(fmt.Sprintf("server starting at :%d", listenPort))
	err = http.ListenAndServe(fmt.Sprintf(":%d", listenPort), nil)
	if err != nil {
		logger.Println(err.Error())
	}
}
