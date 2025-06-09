package server

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/openx/openx-enrichment-service-template/pkg/config"
	"github.com/openx/openx-enrichment-service-template/pkg/metrics"
	"github.com/openx/openx-enrichment-service-template/pkg/openrtb"
	"github.com/openx/openx-enrichment-service-template/pkg/storage"

	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config *config.Config
	logger *zap.Logger
	rand   *rand.Rand
	gcs    *storage.Client
}

// New creates a new server instance
func New(config *config.Config, logger *zap.Logger) (*Server, error) {
	// Initialize GCS client if buckets are configured
	var gcsClient *storage.Client
	if config.GCSInboxBucket != "" {
		var err error
		gcsClient, err = storage.NewClient(context.Background(), config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize GCS client: %w", err)
		}

		// Verify bucket access
		ctx := context.Background()
		if _, err := gcsClient.InboxBucket().Attrs(ctx); err != nil {
			return nil, fmt.Errorf("failed to access inbox bucket %q: %w", config.GCSInboxBucket, err)
		}
	}

	return &Server{
		config: config,
		logger: logger,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		gcs:    gcsClient,
	}, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Create a context with timeout for startup operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test GCS access if configured
	if s.gcs != nil {
		if err := s.testGCSAccess(ctx); err != nil {
			return fmt.Errorf("GCS access test failed: %w", err)
		}
	}

	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/openrtb25", s.handleEnrichment)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.Handle("/metrics", metrics.Handler())

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      mux,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	s.logger.Info("Starting server", zap.Int("port", s.config.Port))
	return server.ListenAndServe()
}

// testGCSAccess verifies that we can read from the inbox bucket
func (s *Server) testGCSAccess(ctx context.Context) error {
	// List files in inbox bucket
	files, err := s.gcs.ListFiles(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list files in inbox bucket: %w", err)
	}

	if len(files) > 0 {
		// Try to read the first file
		if _, err := s.gcs.ReadFile(ctx, files[0]); err != nil {
			return fmt.Errorf("failed to read file %q from inbox bucket: %w", files[0], err)
		}
		s.logger.Info("Successfully tested GCS access", zap.String("file", files[0]))
	} else {
		s.logger.Info("Inbox bucket is empty, GCS access verified")
	}

	return nil
}

// simulateLoad adds both latency and CPU load, ensuring CPU load is part of the total latency
func (s *Server) simulateLoad() {
	if !s.config.SimulateLatency && !s.config.SimulateCPULoad {
		s.logger.Debug("No load simulation enabled")
		return
	}

	// Calculate target latency
	targetLatency := time.Duration(0)
	if s.config.SimulateLatency {
		targetLatency = max(0,
			time.Duration(s.rand.NormFloat64()*float64(s.config.LatencyStdDevMs)+
				float64(s.config.LatencyMeanMs))*time.Millisecond)
	}

	// Calculate CPU load duration as a portion of the total latency
	cpuLoadDuration := time.Duration(0)
	if s.config.SimulateCPULoad && targetLatency > 0 {
		cpuLoadDuration = time.Duration(float64(s.config.CPULoadPercentage)/100.0*float64(targetLatency.Milliseconds())) * time.Millisecond
	}

	// Busy-wait to simulate CPU load for the calculated duration
	cpuStart := time.Now()
	for time.Since(cpuStart) < cpuLoadDuration {
		// Do actual CPU work by computing a hash
		hash := fnv.New64()
		hash.Write([]byte(time.Now().String()))
		_ = hash.Sum64()
	}

	latencyStart := time.Now()
	// Sleep for the remaining time to reach target latency
	remainingTime := targetLatency - time.Since(cpuStart)
	if remainingTime > 0 {
		time.Sleep(remainingTime)
	}

	now := time.Now()
	s.logger.Debug("Simulated load",
		zap.Duration("targetLatency", targetLatency),
		zap.Duration("cpuDuration", latencyStart.Sub(cpuStart)),
		zap.Duration("latencyDuration", now.Sub(latencyStart)),
		zap.Duration("totalDuration", now.Sub(cpuStart)))
}

// ********** EXAMPLE CODE - MUST BE REPLACED **********
// This function demonstrates how to implement enrichment logic.
// You MUST replace this with your actual enrichment implementation.
// The current implementation:
// 1. Simulates CPU load and latency
// 2. Returns a mock response
// DO NOT submit a service that uses this example code!
func (s *Server) exampleEnrichmentLogic(_ context.Context, request openrtb.EnrichmentRequest) (*openrtb.EnrichmentResponse, error) {
	// Simulate load (latency and/or CPU)
	s.simulateLoad()

	// Create response with only allowed fields
	resp := openrtb.EnrichmentResponse{
		ID: request.ID, // Preserve the request ID
		User: &openrtb.EnrichmentUser{
			Data: []openrtb2.Data{
				{
					Name: "segment-provider.com",
					Segment: []openrtb2.Segment{
						{ID: "123"},
					},
				},
			},
			Ext: &openrtb.EnrichmentExt{
				EIDs: []openrtb2.EID{
					{
						Source: "id-provider.com",
						UIDs: []openrtb2.UID{
							{ID: "abc"},
						},
					},
				},
			},
		},
	}
	return &resp, nil
}

func (s *Server) handleEnrichment(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.EnrichmentRequestDuration.Observe(duration.Seconds())

		// Log if request exceeds performance thresholds
		if duration > 10*time.Millisecond {
			s.logger.Warn("Request exceeded p99 latency threshold",
				zap.Duration("duration", duration),
				zap.String("path", r.URL.Path),
			)
		}
		if duration > 30*time.Millisecond {
			s.logger.Error("Request exceeded maximum allowed latency",
				zap.Duration("duration", duration),
				zap.String("path", r.URL.Path),
			)
		}
	}()

	metrics.EnrichmentRequestsTotal.Inc()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		metrics.EnrichmentRequestErrors.WithLabelValues("method_not_allowed").Inc()
		return
	}

	// Check content type
	if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		metrics.EnrichmentRequestErrors.WithLabelValues("invalid_content_type").Inc()
		return
	}

	var request openrtb.EnrichmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		metrics.EnrichmentRequestErrors.WithLabelValues("invalid_request").Inc()
		return
	}

	// Check if we need to return 204 No Content
	if !s.shouldEnrich(request) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ********** EXAMPLE CODE - MUST BE REPLACED **********
	response, err := s.exampleEnrichmentLogic(r.Context(), request)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		metrics.EnrichmentRequestErrors.WithLabelValues("enrichment_error").Inc()
		return
	}

	// Set OpenRTB 2.5 headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-OpenRTB-Version", "2.5")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		metrics.EnrichmentRequestErrors.WithLabelValues("response_encoding_error").Inc()
		return
	}

	metrics.EnrichmentRequestSuccess.Inc()
}

// shouldEnrich determines if we need to enrich the request
func (s *Server) shouldEnrich(request openrtb.EnrichmentRequest) bool {
	// TODO: Implement actual enrichment logic
	// For testing purposes, return false if the request ID contains "no-enrichment"
	return !strings.Contains(request.ID, "no-enrichment")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
