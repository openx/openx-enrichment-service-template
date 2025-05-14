package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandler(t *testing.T) {
	// Record some metrics
	EnrichmentRequestDuration.Observe(0.1)
	EnrichmentRequestsTotal.Inc()
	EnrichmentRequestSuccess.Inc()
	EnrichmentRequestErrors.WithLabelValues("test_error").Inc()

	// Create a request to the metrics endpoint
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	Handler().ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	// Check that all metrics are present
	assert.Contains(t, body, "enrichment_request_duration_seconds")
	assert.Contains(t, body, "enrichment_requests_total")
	assert.Contains(t, body, "enrichment_request_success_total")
	assert.Contains(t, body, "enrichment_request_errors_total{error_type=\"test_error\"}")
}

func TestMetricsRegistration(t *testing.T) {
	// Verify that all metrics are registered with Prometheus
	registry := prometheus.NewRegistry()
	err := registry.Register(EnrichmentRequestDuration)
	require.NoError(t, err, "Failed to register EnrichmentRequestDuration")

	err = registry.Register(EnrichmentRequestsTotal)
	require.NoError(t, err, "Failed to register EnrichmentRequestsTotal")

	err = registry.Register(EnrichmentRequestSuccess)
	require.NoError(t, err, "Failed to register EnrichmentRequestSuccess")

	err = registry.Register(EnrichmentRequestErrors)
	require.NoError(t, err, "Failed to register EnrichmentRequestErrors")
}
