package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fekuna/omnipos-audit-service/config"
	"github.com/fekuna/omnipos-audit-service/internal/audit/handler"
	"github.com/fekuna/omnipos-audit-service/internal/audit/listener"
	"github.com/fekuna/omnipos-audit-service/internal/audit/repository"
	"github.com/fekuna/omnipos-audit-service/internal/audit/usecase"
	"github.com/fekuna/omnipos-pkg/broker"
	"github.com/fekuna/omnipos-pkg/database/mongodb"
	"github.com/fekuna/omnipos-pkg/logger"
	auditv1 "github.com/fekuna/omnipos-proto/proto/audit/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. Load Configuration
	cfg := config.LoadEnv()

	// 2. Initialize Logger
	logConfig := &logger.ZapLoggerConfig{
		IsDevelopment:     false,
		Encoding:          "json",
		Level:             "info",
		DisableCaller:     false,
		DisableStacktrace: false,
	}

	if cfg.AppEnv == "development" {
		logConfig.IsDevelopment = true
		logConfig.Encoding = "console"
		logConfig.Level = "debug"
	}

	appLogger := logger.NewZapLogger(logConfig)
	defer appLogger.Sync()

	// 3. Connect to MongoDB
	mongoCfg := &mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
	}
	mongoClient, err := mongodb.NewClient(mongoCfg)
	if err != nil {
		appLogger.Fatal("Could not connect to MongoDB", zap.Error(err))
	}
	defer mongoClient.Close(nil)
	appLogger.Info("Connected to MongoDB", zap.String("db_name", cfg.MongoDB.Database))

	// 4. Initialize Components
	repo := repository.NewMongoRepository(mongoClient)
	uc := usecase.NewAuditUseCase(repo, appLogger)
	h := handler.NewAuditHandler(uc, appLogger)

	// 5. Initialize Kafka Consumer (if brokers are configured)
	var auditListener *listener.AuditListener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(cfg.Kafka.Brokers) > 0 && cfg.Kafka.Brokers[0] != "" {
		kafkaCfg := &broker.Config{
			Brokers: cfg.Kafka.Brokers,
			Topic:   cfg.Kafka.Topic,
			GroupID: cfg.Kafka.GroupID,
		}
		consumer := broker.NewConsumer(kafkaCfg)
		auditListener = listener.NewAuditListener(consumer, uc, appLogger)

		// Start Kafka listener in background
		go auditListener.Start(ctx)
		appLogger.Info("Kafka Audit Listener started",
			zap.Strings("brokers", cfg.Kafka.Brokers),
			zap.String("topic", cfg.Kafka.Topic),
			zap.String("group_id", cfg.Kafka.GroupID),
		)
	} else {
		appLogger.Warn("Kafka not configured, Audit Listener disabled")
	}

	// 6. Start gRPC Server
	port := cfg.Server.GRPCPort
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	// Register Services
	auditv1.RegisterAuditServiceServer(grpcServer, h)

	// Register Reflection
	reflection.Register(grpcServer)

	appLogger.Info("Starting Audit Service gRPC server", zap.String("port", port))

	// Start gRPC server in background
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			appLogger.Fatal("failed to serve", zap.Error(err))
		}
	}()

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Stop Kafka listener
	cancel()
	if auditListener != nil {
		if err := auditListener.Close(); err != nil {
			appLogger.Error("Failed to close Kafka consumer", zap.Error(err))
		}
	}

	// Stop gRPC server
	grpcServer.GracefulStop()
	appLogger.Info("Server stopped")
}
