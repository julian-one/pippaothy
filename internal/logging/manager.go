package logging

import (
	"encoding/json"
	"log/slog"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds log manager configuration.
type Config struct {
	FilePath   string
	MaxSizeMB  int
	MaxBackups int
	Compress   bool
}

// Manager handles log file writing with rotation and broadcasting.
type Manager struct {
	writer      *lumberjack.Logger
	broadcaster *Broadcaster
	mu          sync.Mutex
}

// NewManager creates a new log manager.
func NewManager(cfg Config, broadcaster *Broadcaster) *Manager {
	writer := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	return &Manager{
		writer:      writer,
		broadcaster: broadcaster,
	}
}

// Write implements io.Writer for the slog JSON handler.
func (m *Manager) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n, err = m.writer.Write(p)
	if err != nil {
		return n, err
	}

	var entry LogEntry
	if json.Unmarshal(p, &entry) == nil {
		m.broadcaster.Broadcast(&entry)
	}

	return n, nil
}

// NewLogger creates an slog.Logger backed by this manager.
func (m *Manager) NewLogger() *slog.Logger {
	handler := slog.NewJSONHandler(m, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(handler)
}

// GetFilePath returns the current log file path.
func (m *Manager) GetFilePath() string {
	return m.writer.Filename
}

// Close closes the log writer.
func (m *Manager) Close() error {
	return m.writer.Close()
}
