package listener

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fekuna/omnipos-audit-service/internal/audit/usecase"
	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/logger"
	"go.uber.org/zap"
)

// AuditListener listens to Kafka for audit events from all services
type AuditListener struct {
	consumer *broker.KafkaConsumer
	uc       usecase.UseCase
	logger   logger.ZapLogger
}

// NewAuditListener creates a new audit listener
func NewAuditListener(consumer *broker.KafkaConsumer, uc usecase.UseCase, logger logger.ZapLogger) *AuditListener {
	return &AuditListener{
		consumer: consumer,
		uc:       uc,
		logger:   logger,
	}
}

// AuditEvent represents an audit event from Kafka
type AuditEvent struct {
	EventID       string       `json:"event_id"`
	EventType     string       `json:"event_type"`
	SourceService string       `json:"source_service"`
	Payload       AuditPayload `json:"payload"`
	Timestamp     time.Time    `json:"timestamp"`
}

// AuditPayload contains the actual audit log data
type AuditPayload struct {
	MerchantID string                 `json:"merchant_id"`
	UserID     string                 `json:"user_id"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Details    map[string]interface{} `json:"details"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	// Enhanced fields
	StoreID       string                 `json:"store_id,omitempty"`
	SessionID     string                 `json:"session_id,omitempty"`
	OldValue      map[string]interface{} `json:"old_value,omitempty"`
	NewValue      map[string]interface{} `json:"new_value,omitempty"`
	Result        string                 `json:"result,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Severity      string                 `json:"severity,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	DurationMs    int64                  `json:"duration_ms,omitempty"`
}

// Start begins listening for audit events from Kafka
func (l *AuditListener) Start(ctx context.Context) {
	l.logger.Info("Starting Audit Kafka Listener", zap.String("topic", "system.audit"))
	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Stopping Audit Kafka Listener")
			return
		default:
			msg, err := l.consumer.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				l.logger.Error("Failed to read kafka message", zap.Error(err))
				time.Sleep(1 * time.Second)
				continue
			}
			l.processMessage(ctx, msg.Value)
		}
	}
}

// processMessage handles a single audit event message
func (l *AuditListener) processMessage(ctx context.Context, value []byte) {
	var event AuditEvent
	if err := json.Unmarshal(value, &event); err != nil {
		l.logger.Error("Failed to unmarshal audit event", zap.Error(err), zap.String("raw", string(value)))
		return
	}

	l.logger.Info("Processing audit event",
		zap.String("event_id", event.EventID),
		zap.String("action", event.Payload.Action),
		zap.String("source", event.SourceService),
	)

	// Create audit log using the usecase with all enhanced fields
	input := &usecase.CreateAuditLogInput{
		MerchantID: event.Payload.MerchantID,
		UserID:     event.Payload.UserID,
		Action:     event.Payload.Action,
		Entity:     event.Payload.EntityType,
		EntityID:   event.Payload.EntityID,
		Details:    event.Payload.Details,
		IPAddress:  event.Payload.IPAddress,
		UserAgent:  event.Payload.UserAgent,
		// Enhanced fields
		StoreID:       event.Payload.StoreID,
		SessionID:     event.Payload.SessionID,
		OldValue:      event.Payload.OldValue,
		NewValue:      event.Payload.NewValue,
		Result:        event.Payload.Result,
		ErrorMessage:  event.Payload.ErrorMessage,
		Severity:      event.Payload.Severity,
		SourceService: event.SourceService, // Source from event level
		CorrelationID: event.Payload.CorrelationID,
		DurationMs:    event.Payload.DurationMs,
	}

	err := l.uc.CreateAuditLog(ctx, input)
	if err != nil {
		l.logger.Error("Failed to create audit log from event",
			zap.Error(err),
			zap.String("event_id", event.EventID),
		)
		return
	}

	l.logger.Info("Audit log created from Kafka event", zap.String("event_id", event.EventID))
}

// Close closes the Kafka consumer
func (l *AuditListener) Close() error {
	return l.consumer.Close()
}
