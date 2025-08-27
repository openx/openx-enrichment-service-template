package server

import (
	"context"
	"hash/fnv"
	"math/rand"
	"time"

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
	return &resp, nil
}

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
