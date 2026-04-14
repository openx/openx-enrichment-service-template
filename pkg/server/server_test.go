package server

// These tests validate basic compliance with the OpenXBuild Enrichment Services specification.
// See SPECIFICATION.md for the complete API contract and requirements.
// The tests cover:
// - HTTP endpoint behavior (/openrtb25, /healthz, /metrics)
// - Request validation (content type, method, JSON format)
// - Response validation (status codes, headers, body format)
// - Performance monitoring (metrics collection and exposure)

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agenticv1 "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/bidstream/mutation/v1"
	agenticsvc "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/bidstream/mutation/services/v1"
	openrtbv2 "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/openrtb/v2"
	"github.com/openx/openx-enrichment-service-template/pkg/config"
	"github.com/openx/openx-enrichment-service-template/pkg/metrics"
	"github.com/openx/openx-enrichment-service-template/pkg/openrtb"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
				// Verify deal enrichment is present when impressions are in the request
				assert.NotNil(t, resp.Imp)
				assert.Len(t, resp.Imp, 1)
				assert.NotNil(t, resp.Imp[0].PMP)
				assert.NotEmpty(t, resp.Imp[0].PMP.Deals)
				assert.Len(t, resp.Imp[0].PMP.Deals, 2)
				assert.Equal(t, "OX-qav-NmrU2e", resp.Imp[0].PMP.Deals[0].ID)
				assert.Equal(t, 0.10, resp.Imp[0].PMP.Deals[0].BidFloor)
				assert.Equal(t, "USD", resp.Imp[0].PMP.Deals[0].BidFloorCur)
				assert.Equal(t, "OX-qav-ZrOLsj", resp.Imp[0].PMP.Deals[1].ID)
				assert.Equal(t, 2.50, resp.Imp[0].PMP.Deals[1].BidFloor)
				assert.Equal(t, "USD", resp.Imp[0].PMP.Deals[1].BidFloorCur)
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
	cfg := &config.Config{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	for _, path := range []string{"/healthz", "/health/ready", "/health/live"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rr := httptest.NewRecorder()
			srv.handleHealth(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "OK", rr.Body.String())
		})
	}
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

func TestServer_AnalyzeRequest(t *testing.T) {
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

	tests := []struct {
		name        string
		request     openrtb.EnrichmentRequest
		expectedID  string
		description string
	}{
		{
			name: "Base segment for empty request",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
			},
			expectedID:  "base-segment",
			description: "Should return base segment when no specific objects are found",
		},
		{
			name: "Banner detection",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
				},
			},
			expectedID:  "banner-detected",
			description: "Should detect banner objects and return banner-detected",
		},
		{
			name: "Video detection",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Video: &openrtb2.Video{},
					},
				},
			},
			expectedID:  "video-detected",
			description: "Should detect video objects and return video-detected",
		},
		{
			name: "Native detection",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Native: &openrtb2.Native{},
					},
				},
			},
			expectedID:  "native-detected",
			description: "Should detect native objects and return native-detected",
		},
		{
			name: "Audio detection",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Audio: &openrtb2.Audio{},
					},
				},
			},
			expectedID:  "audio-detected",
			description: "Should detect audio objects and return audio-detected",
		},
		{
			name: "Banner and video mixed",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
					{
						ID:    "imp2",
						Video: &openrtb2.Video{},
					},
				},
			},
			expectedID:  "banner-video-mixed",
			description: "Should detect mixed banner and video and return banner-video-mixed",
		},
		{
			name: "Multiple formats mixed",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
					{
						ID:     "imp2",
						Native: &openrtb2.Native{},
					},
				},
			},
			expectedID:  "mixed-formats",
			description: "Should detect mixed formats and return mixed-formats",
		},
		{
			name: "Mobile device with banner",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 1, // Mobile
				},
			},
			expectedID:  "banner-detected-mobile",
			description: "Should detect banner on mobile device and append device type",
		},
		{
			name: "Desktop device with video",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Video: &openrtb2.Video{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 2, // Personal Computer
				},
			},
			expectedID:  "video-detected-desktop",
			description: "Should detect video on desktop device and append device type",
		},
		{
			name: "Connected TV with native",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Native: &openrtb2.Native{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 3, // Connected TV
				},
			},
			expectedID:  "native-detected-ctv",
			description: "Should detect native on connected TV and append device type",
		},
		{
			name: "Phone with audio",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						Audio: &openrtb2.Audio{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 4, // Phone
				},
			},
			expectedID:  "audio-detected-phone",
			description: "Should detect audio on phone and append device type",
		},
		{
			name: "Tablet with mixed formats",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
					{
						ID:    "imp2",
						Video: &openrtb2.Video{},
					},
					{
						ID:     "imp3",
						Native: &openrtb2.Native{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 5, // Tablet
				},
			},
			expectedID:  "mixed-formats-tablet",
			description: "Should detect mixed formats on tablet and append device type",
		},
		{
			name: "Unknown device type",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
				},
				Device: &openrtb2.Device{
					DeviceType: 99, // Unknown device type
				},
			},
			expectedID:  "banner-detected",
			description: "Should detect banner but not append unknown device type",
		},
		{
			name: "Device without device type",
			request: openrtb.EnrichmentRequest{
				ID: "test-id",
				Imp: []openrtb2.Imp{
					{
						ID:     "imp1",
						Banner: &openrtb2.Banner{},
					},
				},
				Device: &openrtb2.Device{
					// No DeviceType specified
				},
			},
			expectedID:  "banner-detected",
			description: "Should detect banner but not append device type when not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.analyzer.AnalyzeRequest(&tt.request)
			assert.Equal(t, tt.expectedID, result, tt.description)
		})
	}
}

func TestServer_AnalyzeBidRequest_ARTF(t *testing.T) {
	cfg := &config.Config{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	ptrStr := func(s string) *string { return &s }
	ptrInt32 := func(i int32) *int32 { return &i }

	tests := []struct {
		name        string
		request     *openrtbv2.BidRequest
		expectedID  string
		description string
	}{
		{
			name:        "Base segment for empty request",
			request:     &openrtbv2.BidRequest{Id: ptrStr("test-id")},
			expectedID:  "base-segment",
			description: "Should return base segment when no specific objects are found",
		},
		{
			name:        "Banner detection",
			request:     &openrtbv2.BidRequest{Id: ptrStr("test-id"), Imp: []*openrtbv2.BidRequest_Imp{{Id: ptrStr("imp1"), Banner: &openrtbv2.BidRequest_Banner{}}}},
			expectedID:  "banner-detected",
			description: "Should detect banner objects and return banner-detected",
		},
		{
			name:        "Video detection",
			request:     &openrtbv2.BidRequest{Id: ptrStr("test-id"), Imp: []*openrtbv2.BidRequest_Imp{{Id: ptrStr("imp1"), Video: &openrtbv2.BidRequest_Video{}}}},
			expectedID:  "video-detected",
			description: "Should detect video objects and return video-detected",
		},
		{
			name:        "Mobile device with banner",
			request:     &openrtbv2.BidRequest{Id: ptrStr("test-id"), Imp: []*openrtbv2.BidRequest_Imp{{Id: ptrStr("imp1"), Banner: &openrtbv2.BidRequest_Banner{}}}, Device: &openrtbv2.BidRequest_Device{Devicetype: ptrInt32(1)}},
			expectedID:  "banner-detected-mobile",
			description: "Should detect banner on mobile device and append device type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := srv.artfAnalyzer.AnalyzeBidRequest(tt.request)
			assert.Equal(t, tt.expectedID, result, tt.description)
		})
	}
}

func TestServer_GetMutations_ARTF(t *testing.T) {
	cfg := &config.Config{
		Port:         8080,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  5 * time.Second,
	}
	logger, _ := zap.NewDevelopment()
	srv, err := New(cfg, logger)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer lis.Close()

	gs := grpc.NewServer()
	registerRTBExtensionPoint(gs, logger, srv.ExampleARTFEnrichmentLogic)
	go func() { _ = gs.Serve(lis) }()
	defer gs.Stop()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := agenticsvc.NewRTBExtensionPointClient(conn)
	ctx := context.Background()

	ptrStr := func(s string) *string { return &s }
	req := &agenticv1.RTBRequest{
		Id: ptrStr("grpc-test-id"),
		BidRequest: &openrtbv2.BidRequest{
			Id:  ptrStr("grpc-test-id"),
			Imp: []*openrtbv2.BidRequest_Imp{{Id: ptrStr("imp1"), Banner: &openrtbv2.BidRequest_Banner{}}},
		},
	}

	resp, err := client.GetMutations(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "grpc-test-id", resp.GetId())
	require.NotEmpty(t, resp.Mutations)

	// Verify segment mutation uses IAB path format
	var segmentMutation *agenticv1.Mutation
	for _, m := range resp.Mutations {
		if m.GetIntent() == agenticv1.Intent_ACTIVATE_SEGMENTS {
			segmentMutation = m
			break
		}
	}
	require.NotNil(t, segmentMutation, "expected ACTIVATE_SEGMENTS mutation")
	assert.Equal(t, "/user/data/segment", segmentMutation.GetPath())
	assert.Equal(t, agenticv1.Operation_OPERATION_ADD, segmentMutation.GetOp())
	assert.NotEmpty(t, segmentMutation.GetIds().GetId())

	// Verify deal floor mutations use IAB path format: /imp/{impId}/pmp/deals/{dealId}
	var dealMutations []*agenticv1.Mutation
	for _, m := range resp.Mutations {
		if m.GetIntent() == agenticv1.Intent_ADJUST_DEAL_FLOOR {
			dealMutations = append(dealMutations, m)
		}
	}
	require.NotEmpty(t, dealMutations, "expected ADJUST_DEAL_FLOOR mutations")
	for _, m := range dealMutations {
		assert.Regexp(t, `^/imp/[^/]+/pmp/deals/[^/]+$`, m.GetPath(), "deal path should be /imp/{impId}/pmp/deals/{dealId}")
		assert.Equal(t, agenticv1.Operation_OPERATION_REPLACE, m.GetOp())
		assert.NotNil(t, m.GetAdjustDeal())
	}
}
