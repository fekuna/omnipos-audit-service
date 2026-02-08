package usecase

import (
	"context"
	"time"

	"github.com/fekuna/omnipos-audit-service/internal/audit/repository"
	"github.com/fekuna/omnipos-pkg/logger"
	"github.com/google/uuid"
)

type UseCase interface {
	CreateAuditLog(ctx context.Context, input *CreateAuditLogInput) error
	ListAuditLogs(ctx context.Context, input *ListAuditLogsInput) ([]repository.AuditLog, int32, error)
}

type CreateAuditLogInput struct {
	MerchantID string
	UserID     string
	Action     string
	Entity     string
	EntityID   string
	Details    map[string]interface{}
	IPAddress  string
	UserAgent  string
	// Enhanced fields
	StoreID       string
	SessionID     string
	OldValue      map[string]interface{}
	NewValue      map[string]interface{}
	Result        string
	ErrorMessage  string
	Severity      string
	SourceService string
	CorrelationID string
	DurationMs    int64
}

type ListAuditLogsInput struct {
	MerchantID string
	UserID     string
	Entity     string
	EntityID   string
	Action     string
	StartDate  time.Time
	EndDate    time.Time
	Page       int32
	PageSize   int32
	// Enhanced filters
	StoreID       string
	Severity      string
	Result        string
	SourceService string
	CorrelationID string
}

type auditUseCase struct {
	repo   repository.Repository
	logger logger.ZapLogger
}

func NewAuditUseCase(repo repository.Repository, logger logger.ZapLogger) UseCase {
	return &auditUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *auditUseCase) CreateAuditLog(ctx context.Context, input *CreateAuditLogInput) error {
	// Set defaults for required fields
	result := input.Result
	if result == "" {
		result = "success"
	}
	severity := input.Severity
	if severity == "" {
		severity = "info"
	}

	log := &repository.AuditLog{
		ID:         uuid.New().String(),
		MerchantID: input.MerchantID,
		UserID:     input.UserID,
		Action:     input.Action,
		Entity:     input.Entity,
		EntityID:   input.EntityID,
		Details:    input.Details,
		IPAddress:  input.IPAddress,
		UserAgent:  input.UserAgent,
		Timestamp:  time.Now(),
		// Enhanced fields
		StoreID:       input.StoreID,
		SessionID:     input.SessionID,
		OldValue:      input.OldValue,
		NewValue:      input.NewValue,
		Result:        result,
		ErrorMessage:  input.ErrorMessage,
		Severity:      severity,
		SourceService: input.SourceService,
		CorrelationID: input.CorrelationID,
		DurationMs:    input.DurationMs,
	}

	return uc.repo.CreateAuditLog(ctx, log)
}

func (uc *auditUseCase) ListAuditLogs(ctx context.Context, input *ListAuditLogsInput) ([]repository.AuditLog, int32, error) {
	filter := map[string]interface{}{
		"merchant_id": input.MerchantID,
		"user_id":     input.UserID,
		"entity":      input.Entity,
		"entity_id":   input.EntityID,
		"action":      input.Action,
		"start_date":  input.StartDate,
		"end_date":    input.EndDate,
		// Enhanced filters
		"store_id":       input.StoreID,
		"severity":       input.Severity,
		"result":         input.Result,
		"source_service": input.SourceService,
		"correlation_id": input.CorrelationID,
	}

	return uc.repo.ListAuditLogs(ctx, filter, input.Page, input.PageSize)
}
