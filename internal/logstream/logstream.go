package logstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

type LogEntry struct {
	Time    string         `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"msg"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

type FileLogger struct {
	handler  slog.Handler
	logFile  *os.File
	fileMu   sync.Mutex
	filePath string
}

func NewFileLogger(handler slog.Handler, logFilePath string) *FileLogger {
	fl := &FileLogger{
		handler:  handler,
		filePath: logFilePath,
	}

	if logFilePath != "" {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			fl.logFile = f
		}
	}

	return fl
}

func (fl *FileLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return fl.handler.Enabled(ctx, level)
}

func (fl *FileLogger) Handle(ctx context.Context, r slog.Record) error {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	entry := LogEntry{
		Time:    r.Time.Format(time.RFC3339),
		Level:   r.Level.String(),
		Message: r.Message,
	}
	if len(attrs) > 0 {
		entry.Attrs = attrs
	}

	if fl.logFile != nil {
		if data, err := json.Marshal(entry); err == nil {
			fl.fileMu.Lock()
			fl.logFile.Write(append(data, '\n'))
			fl.fileMu.Unlock()
		}
	}

	return fl.handler.Handle(ctx, r)
}

func (fl *FileLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &FileLogger{
		handler:  fl.handler.WithAttrs(attrs),
		logFile:  fl.logFile,
		filePath: fl.filePath,
	}
}

func (fl *FileLogger) WithGroup(name string) slog.Handler {
	return &FileLogger{
		handler:  fl.handler.WithGroup(name),
		logFile:  fl.logFile,
		filePath: fl.filePath,
	}
}

// ReadHistory reads logs from file (most recent N entries)
func (fl *FileLogger) ReadHistory(limit int) ([]LogEntry, error) {
	if fl.filePath == "" {
		return []LogEntry{}, nil
	}

	file, err := os.Open(fl.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// StreamLogs tails the log file and writes SSE events
func (fl *FileLogger) StreamLogs(ctx context.Context, w io.Writer, flusher func()) error {
	if fl.filePath == "" {
		// No file configured, just keep connection open
		<-ctx.Done()
		return nil
	}

	// Wait for file to exist
	var file *os.File
	var err error
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		file, err = os.Open(fl.filePath)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}
	defer file.Close()

	// Start from end of file (only new logs)
	file.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(file)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadBytes('\n')
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if err != nil {
				return err
			}

			// Trim trailing newline since SSE format adds its own
			line = bytes.TrimSuffix(line, []byte("\n"))
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher()
		}
	}
}
