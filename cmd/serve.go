package cmd

import (
	"log/slog"
	"net/http"
	"os"

	"pippaothy/internal/database"
	"pippaothy/route"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long:  `Start the Pippaothy web server on port 8080`,
	Run:   runServe,
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

	// Initialize routes
	mux := route.Initialize(db.DB, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start server
	logger.Info("Starting server on port 8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}