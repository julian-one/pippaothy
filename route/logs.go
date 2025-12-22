package route

import (
	"net/http"
	"strconv"

	"pippaothy/internal/logstream"
	"pippaothy/internal/middleware"
)

// GetLogHistory returns historical logs as JSON array
func GetLogHistory(fileLogger *logstream.FileLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("get log history handler started")

		limit := 1000
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		log.Info("reading log history", "limit", limit)

		entries, err := fileLogger.ReadHistory(limit)
		if err != nil {
			log.Error("failed to read log history", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to read logs")
			return
		}

		log.Info("get log history handler completed successfully", "count", len(entries))
		writeJSON(w, http.StatusOK, entries)
	}
}

// StreamLogs streams new log entries via SSE
func StreamLogs(fileLogger *logstream.FileLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("stream logs handler started")

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Error("streaming not supported by response writer")
			writeError(w, http.StatusInternalServerError, "Streaming not supported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		log.Info("SSE connection established")
		w.Write([]byte(": connected\n\n"))
		flusher.Flush()

		if err := fileLogger.StreamLogs(r.Context(), w, flusher.Flush); err != nil {
			log.Info("stream logs connection closed", "error", err)
			return
		}
		log.Info("stream logs handler completed")
	}
}
