package server

import (
	"context"
	"hash/fnv"
	"math/rand"
	"time"

	agenticv1 "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/bidstream/mutation/v1"
	openrtbv2 "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/openrtb/v2"
	"github.com/openx/openx-enrichment-service-template/pkg/openrtb"

	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
)

// ********** EXAMPLE CODE - DO NOT USE IN PRODUCTION **********
// ExampleEnrichmentLogic demonstrates how to implement enrichment logic.
// This is placeholder code that should be replaced with your actual implementation.
func (s *Server) ExampleEnrichmentLogic(ctx context.Context, request *openrtb.EnrichmentRequest) (*openrtb.EnrichmentResponse, error) {
	// Simulate load (latency and/or CPU)
	s.simulateLoad()

	// Analyze the request to determine segment ID
	segmentID := s.analyzer.AnalyzeRequest(request)

	// Create response with only allowed fields
	resp := openrtb.EnrichmentResponse{
		ID: request.ID, // Preserve the request ID
		User: &openrtb.EnrichmentUser{
			Data: []openrtb2.Data{
				{
					Name: "segment-provider.com",
					Segment: []openrtb2.Segment{
						{ID: segmentID},
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

	// Add a deal floor override to the first impression if impressions are present
	if len(request.Imp) > 0 {
		imp := request.Imp[0]
		resp.Imp = []openrtb.EnrichmentImp{
			{
				ID: imp.ID,
				PMP: &openrtb.EnrichmentPMP{
					Deals: []openrtb2.Deal{
						{
							ID:          "OX-qav-NmrU2e",
							BidFloor:    0.10,
							BidFloorCur: "USD",
						},
						{
							ID:          "OX-qav-ZrOLsj",
							BidFloor:    2.50,
							BidFloorCur: "USD",
						},
					},
				},
			},
		}
	}

	return &resp, nil
}

// ExampleARTFEnrichmentLogic demonstrates how to implement enrichment logic for the IAB ARTF GetMutations endpoint.
// Takes an IAB RTBRequest (proto types), returns an RTBResponse with mutations.
func (s *Server) ExampleARTFEnrichmentLogic(ctx context.Context, req *agenticv1.RTBRequest) (*agenticv1.RTBResponse, error) {
	s.simulateLoad()

	br := req.GetBidRequest()
	id := req.GetId()
	if br != nil && br.Id != nil {
		id = *br.Id
	}

	segmentID := s.artfAnalyzer.AnalyzeBidRequest(br)

	out := &agenticv1.RTBResponse{Id: &id}

	out.Mutations = append(out.Mutations, &agenticv1.Mutation{
		Intent: ptr(agenticv1.Intent_ACTIVATE_SEGMENTS),
		Op:     ptr(agenticv1.Operation_OPERATION_ADD),
		Path:   ptr("/user/data/segment"),
		Value:  &agenticv1.Mutation_Ids{Ids: &agenticv1.IDsPayload{Id: []string{segmentID}}},
	})

	out.Mutations = append(out.Mutations, &agenticv1.Mutation{
		Intent: ptr(agenticv1.Intent_INTENT_UNSPECIFIED),
		Op:     ptr(agenticv1.Operation_OPERATION_ADD),
		Path:   ptr("/user/eids"),
		Value:  &agenticv1.Mutation_Ids{Ids: &agenticv1.IDsPayload{Id: []string{"abc"}}},
	})

	if br != nil && len(br.GetImp()) > 0 {
		impID := br.GetImp()[0].GetId()
		for _, deal := range []struct {
			id    string
			floor float64
		}{
			{"OX-qav-NmrU2e", 0.10},
			{"OX-qav-ZrOLsj", 2.50},
		} {
			floor := deal.floor
			path := "/imp/" + impID + "/pmp/deals/" + deal.id
			out.Mutations = append(out.Mutations, &agenticv1.Mutation{
				Intent: ptr(agenticv1.Intent_ADJUST_DEAL_FLOOR),
				Op:     ptr(agenticv1.Operation_OPERATION_REPLACE),
				Path:   &path,
				Value: &agenticv1.Mutation_AdjustDeal{
					AdjustDeal: &agenticv1.AdjustDealPayload{
						Bidfloor: &floor,
						Margin:   &agenticv1.Margin{Value: ptr(0.0), CalculationType: ptr(agenticv1.Margin_CPM)},
					},
				},
			})
		}
	}

	return out, nil
}

func ptr[T any](v T) *T { return &v }

// RequestAnalyzer analyzes OpenRTB requests and determines appropriate segment IDs
// This is example code for demonstration purposes only.
type RequestAnalyzer struct {
	logger *zap.Logger
}

// NewRequestAnalyzer creates a new request analyzer
// This is example code for demonstration purposes only.
func NewRequestAnalyzer(logger *zap.Logger) *RequestAnalyzer {
	return &RequestAnalyzer{
		logger: logger,
	}
}

// AnalyzeRequest examines the OpenRTB request and returns a segment ID that reflects what was found
// This is example code for demonstration purposes only.
func (ra *RequestAnalyzer) AnalyzeRequest(request *openrtb.EnrichmentRequest) string {
	// Start with a base segment ID
	segmentID := "base-segment"

	// Check if request has banner objects
	if request.Imp != nil {
		for _, imp := range request.Imp {
			if imp.Banner != nil {
				segmentID = "banner-detected"
				ra.logger.Debug("Banner object detected in request",
					zap.String("request_id", request.ID),
					zap.String("segment_id", segmentID))
				break
			}
		}
	}

	// Check for video objects
	if request.Imp != nil {
		for _, imp := range request.Imp {
			if imp.Video != nil {
				if segmentID == "banner-detected" {
					segmentID = "banner-video-mixed"
				} else {
					segmentID = "video-detected"
				}
				ra.logger.Debug("Video object detected in request",
					zap.String("request_id", request.ID),
					zap.String("segment_id", segmentID))
			}
		}
	}

	// Check for native objects
	if request.Imp != nil {
		for _, imp := range request.Imp {
			if imp.Native != nil {
				if segmentID == "banner-detected" || segmentID == "video-detected" || segmentID == "banner-video-mixed" {
					segmentID = "mixed-formats"
				} else {
					segmentID = "native-detected"
				}
				ra.logger.Debug("Native object detected in request",
					zap.String("request_id", request.ID),
					zap.String("segment_id", segmentID))
			}
		}
	}

	// Check for audio objects
	if request.Imp != nil {
		for _, imp := range request.Imp {
			if imp.Audio != nil {
				if segmentID != "base-segment" {
					segmentID = "mixed-formats"
				} else {
					segmentID = "audio-detected"
				}
				ra.logger.Debug("Audio object detected in request",
					zap.String("request_id", request.ID),
					zap.String("segment_id", segmentID))
			}
		}
	}

	// Check for specific device types
	if request.Device != nil {
		switch request.Device.DeviceType {
		case 1: // Mobile
			segmentID += "-mobile"
		case 2: // Personal Computer
			segmentID += "-desktop"
		case 3: // Connected TV
			segmentID += "-ctv"
		case 4: // Phone
			segmentID += "-phone"
		case 5: // Tablet
			segmentID += "-tablet"
		}
	}

	ra.logger.Debug("Request analysis complete",
		zap.String("request_id", request.ID),
		zap.String("final_segment_id", segmentID))

	return segmentID
}

// ARTFRequestAnalyzer analyzes IAB proto bid requests and returns a segment ID.
type ARTFRequestAnalyzer struct {
	logger *zap.Logger
}

// NewARTFRequestAnalyzer creates a new ARTFRequestAnalyzer.
func NewARTFRequestAnalyzer(logger *zap.Logger) *ARTFRequestAnalyzer {
	return &ARTFRequestAnalyzer{logger: logger}
}

// AnalyzeBidRequest examines an IAB proto BidRequest and returns a segment ID.
func (ra *ARTFRequestAnalyzer) AnalyzeBidRequest(br *openrtbv2.BidRequest) string {
	segmentID := "base-segment"
	if br == nil {
		return segmentID
	}
	reqID := br.GetId()

	for _, imp := range br.GetImp() {
		if imp.GetBanner() != nil {
			segmentID = "banner-detected"
			ra.logger.Debug("Banner object detected", zap.String("request_id", reqID), zap.String("segment_id", segmentID))
			break
		}
	}
	for _, imp := range br.GetImp() {
		if imp.GetVideo() != nil {
			if segmentID == "banner-detected" {
				segmentID = "banner-video-mixed"
			} else {
				segmentID = "video-detected"
			}
			ra.logger.Debug("Video object detected", zap.String("request_id", reqID), zap.String("segment_id", segmentID))
		}
	}
	for _, imp := range br.GetImp() {
		if imp.GetNative() != nil {
			if segmentID != "base-segment" {
				segmentID = "mixed-formats"
			} else {
				segmentID = "native-detected"
			}
			ra.logger.Debug("Native object detected", zap.String("request_id", reqID), zap.String("segment_id", segmentID))
		}
	}
	for _, imp := range br.GetImp() {
		if imp.GetAudio() != nil {
			if segmentID != "base-segment" {
				segmentID = "mixed-formats"
			} else {
				segmentID = "audio-detected"
			}
			ra.logger.Debug("Audio object detected", zap.String("request_id", reqID), zap.String("segment_id", segmentID))
		}
	}

	if br.GetDevice() != nil {
		switch br.GetDevice().GetDevicetype() {
		case 1:
			segmentID += "-mobile"
		case 2:
			segmentID += "-desktop"
		case 3:
			segmentID += "-ctv"
		case 4:
			segmentID += "-phone"
		case 5:
			segmentID += "-tablet"
		}
	}

	ra.logger.Debug("Request analysis complete", zap.String("request_id", reqID), zap.String("final_segment_id", segmentID))
	return segmentID
}

// simulateLoad adds both latency and CPU load, ensuring CPU load is part of the total latency
// This is example code for demonstration purposes only.
func (s *Server) simulateLoad() {
	if !s.config.SimulateLatency && !s.config.SimulateCPULoad {
		s.logger.Debug("No load simulation enabled")
		return
	}

	// Calculate target latency
	targetLatency := time.Duration(0)
	if s.config.SimulateLatency {
		targetLatency = max(0,
			time.Duration(rand.NormFloat64()*float64(s.config.LatencyStdDevMs)+
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
