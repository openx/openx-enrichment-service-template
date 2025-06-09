package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/openx/openx-enrichment-service-template/pkg/config"
	"github.com/openx/openx-enrichment-service-template/pkg/server"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.LoadFromEnv()

	// Initialize logger
	loggerConfig := zap.NewProductionConfig()
	if err := loggerConfig.Level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid log level %q: %v\n", cfg.LogLevel, err)
		os.Exit(1)
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down server")
}
