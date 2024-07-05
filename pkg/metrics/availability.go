package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    applicationAvailability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
            Name: "release_service_application_availability_gauge",
            Help: "Release Service application availability gauge",
        },
        []string{"application", "reason", "status"},
    )
)

// RecordAvailabilityData exports the check data to be read by Prometheus
func RecordAvailabilityData(application string, reason string, status string, value float64) {
    applicationAvailability.
        With(prometheus.Labels{
            "application": application,
            "reason": reason,
            "status": status,
        }).Set(value)
}

// init
func init() {
    prometheus.MustRegister(applicationAvailability)
}
