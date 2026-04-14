package server

import (
	"context"

	agenticsvc "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/bidstream/mutation/services/v1"
	agenticv1 "github.com/openx/openx-enrichment-service-template/pkg/gen/com/iabtechlab/bidstream/mutation/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type grpcEnrichmentLogic func(ctx context.Context, req *agenticv1.RTBRequest) (*agenticv1.RTBResponse, error)

// rtbExtensionPointServer implements the IAB RTBExtensionPoint gRPC service (GetMutations).
type rtbExtensionPointServer struct {
	agenticsvc.UnimplementedRTBExtensionPointServer
	logger *zap.Logger
	logic  grpcEnrichmentLogic
}

// GetMutations implements RTBExtensionPoint.GetMutations.
func (s *rtbExtensionPointServer) GetMutations(ctx context.Context, req *agenticv1.RTBRequest) (*agenticv1.RTBResponse, error) {
	return s.logic(ctx, req)
}

// registerRTBExtensionPoint registers the RTBExtensionPoint server with the gRPC server.
func registerRTBExtensionPoint(s *grpc.Server, logger *zap.Logger, logic grpcEnrichmentLogic) {
	agenticsvc.RegisterRTBExtensionPointServer(s, &rtbExtensionPointServer{logger: logger, logic: logic})
}
