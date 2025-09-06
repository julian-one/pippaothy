package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"pippaothy/internal/database"
	"pippaothy/internal/server"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pippaothy",
	Short: "Pippaothy - A personal web application",
	Long: `Pippaothy is a personal web application built with Go.
It provides both a web server and other utilities for managing the application`,
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  `Start the Pippaothy web server on port 8080`,
	Run:   runServe,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file found or error loading it", "error", err)
	}

	// Initialize logger
	logger := slog.Default()

	// Initialize database
	db, err := database.NewDB()
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create server
	srv := server.New(db.DB, logger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
		srv.Shutdown(ctx)
	}()

	logger.Info("Starting server on port 8080")
	if err := srv.Start(); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}
