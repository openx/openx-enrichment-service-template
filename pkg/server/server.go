package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/openx/openx-enrichment-service-template/pkg/config"
	"github.com/openx/openx-enrichment-service-template/pkg/metrics"
	"github.com/openx/openx-enrichment-service-template/pkg/openrtb"
	"github.com/openx/openx-enrichment-service-template/pkg/storage"

	"go.uber.org/zap"
)

// Server represents the HTTP server
type Server struct {
	config   *config.Config
	logger   *zap.Logger
	gcs      *storage.Client
	analyzer *RequestAnalyzer
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
		config:   config,
		logger:   logger,
		gcs:      gcsClient,
		analyzer: NewRequestAnalyzer(logger),
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

// ********** EXAMPLE CODE - MUST BE REPLACED **********
// This function calls the example enrichment logic.
func (s *Server) exampleEnrichmentLogic(ctx context.Context, request *openrtb.EnrichmentRequest) (*openrtb.EnrichmentResponse, error) {
	return s.ExampleEnrichmentLogic(ctx, request)
}

func (s *Server) logRequestResponse(request *openrtb.EnrichmentRequest, response *openrtb.EnrichmentResponse) {
	s.logger.Info("OpenRTB request",
		zap.String("request_id", request.ID),
		zap.Any("request", request),
		zap.String("type", "openrtb_request"),
	)
	s.logger.Info("Enrichment response",
		zap.String("request_id", response.ID),
		zap.Any("response", response),
		zap.String("type", "openrtb_response"),
	)
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

	// Check if we should log this request/response pair
	shouldLog := s.config.RequestLogThrottle > 0 && rand.Float64() < s.config.RequestLogThrottle

	// If enrichment is disabled, return 204 No Content
	if s.config.DisableEnrichment {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Check if we need to return 204 No Content
	if !s.shouldEnrich(&request) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// ********** EXAMPLE CODE - MUST BE REPLACED **********
	response, err := s.exampleEnrichmentLogic(r.Context(), &request)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		metrics.EnrichmentRequestErrors.WithLabelValues("enrichment_error").Inc()
		return
	}

	// Log both request and response together if throttling check passed
	if shouldLog {
		s.logRequestResponse(&request, response)
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
func (s *Server) shouldEnrich(request *openrtb.EnrichmentRequest) bool {
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
