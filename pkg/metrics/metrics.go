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
package metrics

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

// CompositeMetric holds instances of GaugeMetric and HistogramMetric
type CompositeMetric struct {
	Gauge     GaugeMetric
	Histogram HistogramMetric
}

// GaugeMetric
type GaugeMetric struct {
	Prefix string
	Labels []string
	Metric *prometheus.GaugeVec
}

// HistogramMetric
type HistogramMetric struct {
	Prefix string
	Labels []string
	Metric *prometheus.HistogramVec
}

// NewGaugeMetric creates a new instance of GaugeMetric
func NewGaugeMetric(prefix string, labels []string) GaugeMetric {
	newGaugeMetric := GaugeMetric{
		Prefix: prefix,
		Labels: labels,
	}

	opts := prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_check_gauge", strings.ToLower(prefix)),
		Help: fmt.Sprintf("%s check_gauge", prefix),
	}
	newGauge := prometheus.NewGaugeVec(opts, labels)
	newGaugeMetric.Metric = newGauge

	return newGaugeMetric
}

// NewHistogramMetric create a new instance of HistogramMetric
func NewHistogramMetric(prefix string, labels []string) HistogramMetric {
	newHistogramMetric := HistogramMetric{
		Prefix: prefix,
		Labels: labels,
	}
	opts := prometheus.HistogramOpts{
		Name: fmt.Sprintf("%s_check_histogram", strings.ToLower(prefix)),
		Help: fmt.Sprintf("%s check_histogram", prefix),
	}
	newHistogram := prometheus.NewHistogramVec(opts, labels)
	newHistogramMetric.Metric = newHistogram

	return newHistogramMetric
}

// Record records a new value for a GaugeMetric
func (gm *GaugeMetric) Record(metadata []string, value float64) {
	// building labels
	labels := map[string]string{}
	for k, v := range gm.Labels {
		labels[v] = metadata[k]
	}
	gm.Metric.With(prometheus.Labels(labels)).Set(value)
}

// Record records a new value for a HistogramMetric
func (hm *HistogramMetric) Record(metadata []string, value float64) {
	// building labels
	labels := map[string]string{}
	for k, v := range hm.Labels {
		labels[v] = metadata[k]
	}
	hm.Metric.With(prometheus.Labels(labels)).Observe(value)
}

// FlipValue flips 0<->1
func FlipValue(value float64) float64 {
	flipped := (int(value) + 1) % 2

	return float64(flipped)
}
