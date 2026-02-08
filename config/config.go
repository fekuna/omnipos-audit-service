package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv string
	Server struct {
		GRPCPort string
	}
	MongoDB struct {
		URI      string
		Database string
	}
	Kafka struct {
		Brokers []string
		Topic   string
		GroupID string
	}
}

func LoadEnv() *Config {
	_ = godotenv.Load() // Ignore error if .env not found (docker env)

	cfg := &Config{}

	cfg.AppEnv = getEnv("APP_ENV", "development")
	cfg.Server.GRPCPort = getEnv("GRPC_PORT", "8086") // Default to 8086 for Audit Service

	cfg.MongoDB.URI = getEnv("MONGODB_URI", "mongodb://localhost:27017")
	cfg.MongoDB.Database = getEnv("MONGODB_DATABASE", "omnipos_audit_db")

	// Kafka consumer configuration
	cfg.Kafka.Brokers = strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ",")
	cfg.Kafka.Topic = getEnv("KAFKA_TOPIC", "system.audit")
	cfg.Kafka.GroupID = getEnv("KAFKA_GROUP_ID", "audit-service-group")

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
