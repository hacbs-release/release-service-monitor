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

type GaugeMetric struct {
	Prefix string
	Labels []string
	Metric *prometheus.GaugeVec
}

func NewGaugeMetric(prefix string, labels []string) GaugeMetric {
	newGaugeMetric := GaugeMetric{
		Prefix: prefix,
		Labels: labels,
	}

	opts := prometheus.GaugeOpts{
		Name: fmt.Sprintf("%s_check", strings.ToLower(prefix)),
		Help: fmt.Sprintf("%s check", prefix),
	}
	newGauge := prometheus.NewGaugeVec(opts, labels)
	newGaugeMetric.Metric = newGauge

	return newGaugeMetric
}

func (gm *GaugeMetric) Record(metadata []string, value float64) {
	// building labels
	labels := map[string]string{}
	for k, v := range gm.Labels {
		labels[v] = metadata[k]
	}
	gm.Metric.With(prometheus.Labels(labels)).Set(value)
}
