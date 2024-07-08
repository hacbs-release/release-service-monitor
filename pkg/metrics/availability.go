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
	"github.com/prometheus/client_golang/prometheus"
)

var (
	applicationAvailability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "release_service_application_availability",
			Help: "Release Service application availability",
		},
		[]string{"application", "reason", "status"},
	)
)

// RecordAvailabilityData exports the check data to be read by Prometheus
func RecordAvailabilityData(application string, reason string, status string, value float64) {
	applicationAvailability.
		With(prometheus.Labels{
			"application": application,
			"reason":      reason,
			"status":      status,
		}).Set(value)
}

// init
func init() {
	prometheus.MustRegister(applicationAvailability)
}