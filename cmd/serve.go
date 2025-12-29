package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"citadel/internal/auth"
	"citadel/internal/cache"
	"citadel/internal/database"
	"citadel/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Citadel HTTP server",
	Long:  `The serve command starts the Citadel HTTP server using the configuration`,
	Run:   runServe,
}

func runServe(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.path", "./citadel.db")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.db", 0)

	// Load configuration
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read config file", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := slog.Default()

	// Initialize database
	db, err := database.New(viper.GetString("database.path"))
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Redis
	cacheConfig := cache.Config{
		Host:     viper.GetString("redis.host"),
		Port:     viper.GetString("redis.port"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}
	rdb, err := cache.New(ctx, cacheConfig)
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Create JWT service from config
	jwtSecret := viper.GetString("jwt.secret")
	if jwtSecret == "" {
		logger.Error("JWT secret not configured")
		os.Exit(1)
	}
	issuer := auth.NewIssuer(jwtSecret)

	// Initialize routes
	routeConfig := route.Config{
		Db:     db,
		Redis:  rdb,
		Issuer: issuer,
		Logger: logger,
	}
	handler := route.Initialize(routeConfig)

	// Create HTTP server
	serverPort := viper.GetString("server.port")
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: handler,
	}

	// Start server
	logger.Info("Starting server", "port", serverPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}
