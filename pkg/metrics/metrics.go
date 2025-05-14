package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// EnrichmentRequestDuration tracks the duration of enrichment requests
	EnrichmentRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "enrichment_request_duration_seconds",
		Help: "Duration of enrichment requests in seconds",
	})

	// EnrichmentRequestsTotal tracks the total number of enrichment requests
	EnrichmentRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "enrichment_requests_total",
		Help: "Total number of enrichment requests",
	})

	// EnrichmentRequestSuccess tracks successful enrichment requests
	EnrichmentRequestSuccess = promauto.NewCounter(prometheus.CounterOpts{
		Name: "enrichment_request_success_total",
		Help: "Total number of successful enrichment requests",
	})

	// EnrichmentRequestErrors tracks enrichment request errors by type
	EnrichmentRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_request_errors_total",
			Help: "Total number of enrichment request errors by type",
		},
		[]string{"error_type"},
	)

	enrichmentRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_requests_total",
			Help: "Total number of enrichment requests",
		},
		[]string{"status"},
	)

	enrichmentLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "enrichment_latency_seconds",
			Help:    "Enrichment request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
)

// Handler returns the Prometheus metrics handler
func Handler() http.Handler {
	return promhttp.Handler()
}
