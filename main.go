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
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hacbs-release/release-availability-metrics/pkg/checks"
	"github.com/hacbs-release/release-availability-metrics/pkg/config"
	"github.com/hacbs-release/release-availability-metrics/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
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
	logger.Printf("Poll interval: %d * time.Second\n", pollInterval)

	// registering metrics
	prefix := cfg.Service.MetricsPrefix
	if prefix == "" {
		prefix = "metrics_server"
	}

	gaugeMetric := metrics.NewGaugeMetric(prefix, []string{"check"})
	histogramMetric := metrics.NewHistogramMetric(prefix, []string{"check", "reason", "status"})

	prometheus.MustRegister(gaugeMetric.Metric)
	prometheus.MustRegister(histogramMetric.Metric)

	metric := metrics.CompositeMetric{
		Gauge:     gaugeMetric,
		Histogram: histogramMetric,
	}

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
			auth := checks.NewQuayAuth(username, password)
			newCheck := checks.NewQuayCheck(
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
			cert := os.Getenv(fmt.Sprintf("%s_HTTP_CERT", strings.ToUpper(httpCheck.Name)))
			key := os.Getenv(fmt.Sprintf("%s_HTTP_KEY", strings.ToUpper(httpCheck.Name)))
			if username == "" {
				username = httpCheck.Username
			}
			if password == "" {
				password = httpCheck.Password
			}
			if cert == "" {
				cert = httpCheck.Cert
			}
			if key == "" {
				key = httpCheck.Key
			}
			newCheck := checks.NewHttpCheck(
				httpCheck.Name,
				username,
				password,
				httpCheck.Url,
				cert,
				key,
				httpCheck.Insecure,
				httpCheck.Follow,
				logger,
				metric)
			_http = append(_http, newCheck)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
		defer ticker.Stop()

		runChecks := func() {
			for _, check := range git {
				check.Check()
			}
			for _, check := range _http {
				check.Check()
			}
			for _, check := range quay {
				check.Check(ctx)
			}
		}

		// Run checks immediately on start
		runChecks()

		for {
			select {
			case <-ctx.Done():
				logger.Println("shutting down check loop")
				return
			case <-ticker.C:
				runChecks()
			}
		}
	}()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfgFilePath := "server-config.yaml"
	if len(os.Args) > 1 {
		cfgFilePath = os.Args[1]
	}

	logger.Printf("loading config from: %s\n", cfgFilePath)
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

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", listenPort),
	}

	go func() {
		logger.Printf("server starting at :%d\n", listenPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Println(err.Error())
		}
	}()

	<-ctx.Done()
	logger.Println("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Printf("server shutdown error: %v\n", err)
	}
	logger.Println("server stopped")
}
