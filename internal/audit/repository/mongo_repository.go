package repository

import (
	"context"
	"time"

	"github.com/fekuna/omnipos-pkg/database/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AuditLog struct {
	ID         string                 `bson:"_id,omitempty"`
	MerchantID string                 `bson:"merchant_id"`
	UserID     string                 `bson:"user_id"`
	Action     string                 `bson:"action"`
	Entity     string                 `bson:"entity"`
	EntityID   string                 `bson:"entity_id"`
	Details    map[string]interface{} `bson:"details"`
	IPAddress  string                 `bson:"ip_address"`
	UserAgent  string                 `bson:"user_agent"`
	Timestamp  time.Time              `bson:"timestamp"`
	// Enhanced fields for best practices
	StoreID       string                 `bson:"store_id,omitempty"`
	SessionID     string                 `bson:"session_id,omitempty"`
	OldValue      map[string]interface{} `bson:"old_value,omitempty"`
	NewValue      map[string]interface{} `bson:"new_value,omitempty"`
	Result        string                 `bson:"result,omitempty"` // success, failure, partial
	ErrorMessage  string                 `bson:"error_message,omitempty"`
	Severity      string                 `bson:"severity,omitempty"` // info, warning, critical
	SourceService string                 `bson:"source_service,omitempty"`
	CorrelationID string                 `bson:"correlation_id,omitempty"`
	DurationMs    int64                  `bson:"duration_ms,omitempty"`
}

type Repository interface {
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, filter map[string]interface{}, page, pageSize int32) ([]AuditLog, int32, error)
}

type mongoRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewMongoRepository(client *mongodb.Client) Repository {
	db := client.Database()
	return &mongoRepository{
		db:         db,
		collection: db.Collection("audit_logs"),
	}
}

func (r *mongoRepository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	_, err := r.collection.InsertOne(ctx, log)
	return err
}

func (r *mongoRepository) ListAuditLogs(ctx context.Context, filter map[string]interface{}, page, pageSize int32) ([]AuditLog, int32, error) {
	// Build BSON filter
	query := bson.M{}
	for k, v := range filter {
		if v != "" {
			query[k] = v
		}
	}

	// Handle date range in filter if present (expecting specific keys)
	if start, ok := filter["start_date"].(time.Time); ok && !start.IsZero() {
		query["timestamp"] = bson.M{"$gte": start}
	}
	if end, ok := filter["end_date"].(time.Time); ok && !end.IsZero() {
		if val, exists := query["timestamp"].(bson.M); exists {
			val["$lte"] = end
			query["timestamp"] = val
		} else {
			query["timestamp"] = bson.M{"$lte": end}
		}
	}

	// Pagination options
	skip := int64((page - 1) * pageSize)
	limit := int64(pageSize)
	opts := options.Find().SetSkip(skip).SetLimit(limit).SetSort(bson.M{"timestamp": -1})

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var logs []AuditLog
	if err = cursor.All(ctx, &logs); err != nil {
		return nil, 0, err
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	return logs, int32(total), nil
}
