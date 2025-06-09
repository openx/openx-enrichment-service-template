package server

// These tests validate basic compliance with the OpenX Hosted RTB Enrichment Services specification.
// See SPECIFICATION.md for the complete API contract and requirements.
// The tests cover:
// - HTTP endpoint behavior (/openrtb25, /healthz, /metrics)
// - Request validation (content type, method, JSON format)
// - Response validation (status codes, headers, body format)
// - Performance monitoring (metrics collection and exposure)

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openx/openx-enrichment-service-template/pkg/config"
	"github.com/openx/openx-enrichment-service-template/pkg/metrics"
	"github.com/openx/openx-enrichment-service-template/pkg/openrtb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestServer_HandleEnrichment(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		Port:            8080,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     5 * time.Second,
		SimulateLatency: false,
		SimulateCPULoad: false,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	tests := []struct {
		name             string
		method           string
		contentType      string
		requestBody      string
		expectedStatus   int
		validateResponse func(t *testing.T, response *http.Response)
	}{
		{
			name:           "Valid request returns 200 with enrichment",
			method:         http.MethodPost,
			contentType:    "application/json",
			requestBody:    `{"id": "test-id", "imp": [{"id": "imp1", "banner": {}}]}`,
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, response *http.Response) {
				assert.Equal(t, "application/json", response.Header.Get("Content-Type"))
				assert.Equal(t, "2.5", response.Header.Get("X-OpenRTB-Version"))

				var resp openrtb.EnrichmentResponse
				err := json.NewDecoder(response.Body).Decode(&resp)
				require.NoError(t, err)

				assert.Equal(t, "test-id", resp.ID)
				assert.NotNil(t, resp.User)
				assert.NotEmpty(t, resp.User.Data)
				assert.NotEmpty(t, resp.User.Ext)
			},
		},
		{
			name:           "GET method returns 405",
			method:         http.MethodGet,
			contentType:    "application/json",
			requestBody:    `{}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid content type returns 400",
			method:         http.MethodPost,
			contentType:    "text/plain",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON returns 400",
			method:         http.MethodPost,
			contentType:    "application/json",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "No enrichment needed returns 204",
			method:         http.MethodPost,
			contentType:    "application/json",
			requestBody:    `{"id": "no-enrichment-test-id", "imp": [{"id": "imp1", "banner": {}}]}`,
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/openrtb25", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", tt.contentType)
			rr := httptest.NewRecorder()

			srv.handleEnrichment(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.validateResponse != nil {
				tt.validateResponse(t, rr.Result())
			}
		})
	}
}

func TestServer_HandleHealth(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	srv.handleHealth(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestServer_HandleMetrics(t *testing.T) {
	// Setup test server
	cfg := &config.Config{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	// First make a request to generate some metrics
	req := httptest.NewRequest(http.MethodPost, "/openrtb25", bytes.NewBufferString(`{"id": "test-id", "imp": [{"id": "imp1", "banner": {}}]}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.handleEnrichment(rr, req)

	// Now check the metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr = httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "enrichment_requests_total")
	assert.Contains(t, rr.Body.String(), "enrichment_request_duration_seconds")
}
