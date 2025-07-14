package main

import (
	"log/slog"
	"os"
	"pippaothy/internal/database"
	"pippaothy/internal/server"
)

func main() {
	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		),
	)

	db, err := database.Create()
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database connection established")

	s := server.New(db, logger)
	if err := s.Start(); err != nil {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}
