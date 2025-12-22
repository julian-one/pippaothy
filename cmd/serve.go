package cmd

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"pippaothy/internal/auth"
	"pippaothy/internal/database"
	"pippaothy/internal/logstream"
	"pippaothy/internal/redis"
	"pippaothy/route"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Pippaothy HTTP server",
	Long:  `The serve command starts the Pippaothy HTTP server using the configuration`,
	Run:   runServe,
}

func runServe(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	// Load configuration
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("logging.file", "/var/log/pippaothy/server.log")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read config file", "error", err)
		os.Exit(1)
	}

	// Initialize logging with file output
	logFilePath := viper.GetString("logging.file")
	fileLogger := logstream.NewFileLogger(slog.Default().Handler(), logFilePath)
	logger := slog.New(fileLogger)

	// Initialize database
	dbConfig := database.Config{
		Host:     viper.GetString("database.host"),
		Port:     viper.GetString("database.port"),
		User:     viper.GetString("database.user"),
		Password: viper.GetString("database.password"),
		Name:     viper.GetString("database.name"),
	}

	db, err := database.New(dbConfig)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize Redis
	redisConfig := redis.Config{
		Host:     viper.GetString("redis.host"),
		Port:     viper.GetString("redis.port"),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	}

	rdb, err := redis.New(ctx, redisConfig)
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
	handler := route.Initialize(route.Config{
		Db:         db,
		Redis:      rdb,
		Issuer:     issuer,
		Logger:     logger,
		FileLogger: fileLogger,
	})

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
