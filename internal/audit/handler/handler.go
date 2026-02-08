package handler

import (
	"context"

	// For model type re-use or DTO mapping

	"github.com/fekuna/omnipos-audit-service/internal/audit/usecase"
	"github.com/fekuna/omnipos-pkg/logger"
	auditv1 "github.com/fekuna/omnipos-proto/proto/audit/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditHandler struct {
	auditv1.UnimplementedAuditServiceServer
	uc     usecase.UseCase
	logger logger.ZapLogger
}

func NewAuditHandler(uc usecase.UseCase, logger logger.ZapLogger) *AuditHandler {
	return &AuditHandler{
		uc:     uc,
		logger: logger,
	}
}

func (h *AuditHandler) CreateAuditLog(ctx context.Context, req *auditv1.CreateAuditLogRequest) (*emptypb.Empty, error) {
	// Extract metadata
	merchantID := ""
	userID := ""
	ipAddress := ""
	userAgent := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if val := md.Get("x-merchant-id"); len(val) > 0 {
			merchantID = val[0]
		}
		if val := md.Get("x-user-id"); len(val) > 0 {
			userID = val[0]
		}
		if val := md.Get("x-forwarded-for"); len(val) > 0 {
			ipAddress = val[0]
		}
		if val := md.Get("user-agent"); len(val) > 0 {
			userAgent = val[0]
		}
	}

	details := make(map[string]interface{})
	if req.Details != nil {
		details = req.Details.AsMap()
	}

	oldValue := make(map[string]interface{})
	if req.OldValue != nil {
		oldValue = req.OldValue.AsMap()
	}

	newValue := make(map[string]interface{})
	if req.NewValue != nil {
		newValue = req.NewValue.AsMap()
	}

	input := &usecase.CreateAuditLogInput{
		MerchantID: merchantID,
		UserID:     userID,
		Action:     req.Action,
		Entity:     req.Entity,
		EntityID:   req.EntityId,
		Details:    details,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		// Enhanced fields
		StoreID:       req.StoreId,
		SessionID:     req.SessionId,
		OldValue:      oldValue,
		NewValue:      newValue,
		Result:        req.Result,
		ErrorMessage:  req.ErrorMessage,
		Severity:      req.Severity,
		SourceService: req.SourceService,
		CorrelationID: req.CorrelationId,
		DurationMs:    req.DurationMs,
	}

	if err := h.uc.CreateAuditLog(ctx, input); err != nil {
		h.logger.Error("Failed to create audit log", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create audit log")
	}

	return &emptypb.Empty{}, nil
}

func (h *AuditHandler) ListAuditLogs(ctx context.Context, req *auditv1.ListAuditLogsRequest) (*auditv1.ListAuditLogsResponse, error) {
	// Audit logs are restricted to the merchant
	merchantID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if val := md.Get("x-merchant-id"); len(val) > 0 {
			merchantID = val[0]
		}
	}

	input := &usecase.ListAuditLogsInput{
		MerchantID: merchantID,
		UserID:     req.UserId,
		Entity:     req.Entity,
		EntityID:   req.EntityId,
		Action:     req.Action,
		Page:       req.Page,
		PageSize:   req.PageSize,
		// Enhanced filters
		StoreID:       req.StoreId,
		Severity:      req.Severity,
		Result:        req.Result,
		SourceService: req.SourceService,
		CorrelationID: req.CorrelationId,
	}

	if req.StartDate != nil {
		input.StartDate = req.StartDate.AsTime()
	}
	if req.EndDate != nil {
		input.EndDate = req.EndDate.AsTime()
	}

	logs, total, err := h.uc.ListAuditLogs(ctx, input)
	if err != nil {
		h.logger.Error("Failed to list audit logs", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list audit logs")
	}

	respLogs := make([]*auditv1.AuditLog, len(logs))
	for i, l := range logs {
		details, _ := structpb.NewStruct(l.Details)
		oldValue, _ := structpb.NewStruct(l.OldValue)
		newValue, _ := structpb.NewStruct(l.NewValue)

		respLogs[i] = &auditv1.AuditLog{
			Id:         l.ID,
			MerchantId: l.MerchantID,
			UserId:     l.UserID,
			Action:     l.Action,
			Entity:     l.Entity,
			EntityId:   l.EntityID,
			Details:    details,
			IpAddress:  l.IPAddress,
			UserAgent:  l.UserAgent,
			Timestamp:  timestamppb.New(l.Timestamp),
			// Enhanced fields
			StoreId:       l.StoreID,
			SessionId:     l.SessionID,
			OldValue:      oldValue,
			NewValue:      newValue,
			Result:        l.Result,
			ErrorMessage:  l.ErrorMessage,
			Severity:      l.Severity,
			SourceService: l.SourceService,
			CorrelationId: l.CorrelationID,
			DurationMs:    l.DurationMs,
		}
	}

	return &auditv1.ListAuditLogsResponse{
		Logs:  respLogs,
		Total: total,
	}, nil
}
