package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"pippaothy/internal/database"
	"pippaothy/internal/logs"
	"pippaothy/internal/server"
	"syscall"
	"time"
)

// main is the application entry point that:
// 1. Configures structured logging to both file and stdout
// 2. Establishes database connection
// 3. Creates and starts the HTTP server
func main() {
	// Setup log file path using the same logic as internal/logs package
	logFilePath := logs.GetLogFilePath()

	// Create log directory if it doesn't exist
	os.MkdirAll(filepath.Dir(logFilePath), 0755)

	// Open log file with create, write, and append flags
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Create multi-writer to write logs to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Configure structured JSON logger with debug level
	logger := slog.New(
		slog.NewJSONHandler(
			multiWriter,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	// Initialize database connection
	db, err := database.Create()
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database connection established")

	// Create HTTP server
	s := server.New(db, logger)

	// Create a channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		logger.Info("Starting HTTP server on :8080")
		if err := s.Start(); err != nil {
			serverErrChan <- err
		}
	}()

	// Wait for either server error or shutdown signal
	select {
	case err := <-serverErrChan:
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig.String())
		
		// Create a context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := s.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown failed", "error", err)
			os.Exit(1)
		}
		
		logger.Info("Server shutdown completed successfully")
	}
}

