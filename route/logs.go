package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"citadel/internal/logging"
	"citadel/internal/middleware"

	"github.com/google/uuid"
)

// LogsStream returns an SSE handler for streaming logs.
func LogsStream(manager *logging.Manager, broadcaster *logging.Broadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := middleware.GetLogger(r)

		filter, err := parseLogFilter(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Error("streaming not supported")
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		subID := uuid.New().String()
		log.Info("sse client connected", "subscriber_id", subID)

		sub := broadcaster.Subscribe(subID, filter)
		defer func() {
			broadcaster.Unsubscribe(subID)
			log.Info("sse client disconnected", "subscriber_id", subID)
		}()

		if shouldReadHistorical(filter) {
			historicalCh := make(chan *logging.LogEntry, 100)
			reader := logging.NewReader(manager.GetFilePath())

			historicalFilter := &logging.Filter{
				Levels:    filter.Levels,
				Search:    filter.Search,
				StartTime: filter.StartTime,
				EndTime:   timePtr(time.Now()),
			}
			if filter.EndTime != nil && filter.EndTime.Before(time.Now()) {
				historicalFilter.EndTime = filter.EndTime
			}

			go func() {
				reader.ReadHistorical(ctx, historicalFilter, historicalCh)
			}()

			for entry := range historicalCh {
				if err := sendSSEEvent(w, flusher, "log", entry); err != nil {
					return
				}
			}

			sendSSEEvent(w, flusher, "marker", map[string]string{
				"type": "historical_complete",
				"time": time.Now().Format(time.RFC3339),
			})
		}

		log.Info("switching to live streaming", "subscriber_id", subID)

		keepalive := time.NewTicker(30 * time.Second)
		defer keepalive.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case entry, ok := <-sub.Ch:
				if !ok {
					return
				}
				if err := sendSSEEvent(w, flusher, "log", entry); err != nil {
					return
				}

			case <-keepalive.C:
				fmt.Fprintf(w, ": keepalive %s\n\n", time.Now().Format(time.RFC3339))
				flusher.Flush()
			}
		}
	}
}

func parseLogFilter(r *http.Request) (*logging.Filter, error) {
	query := r.URL.Query()

	var levels []string
	if levelParam := query.Get("level"); levelParam != "" {
		for _, l := range strings.Split(levelParam, ",") {
			upper := strings.ToUpper(strings.TrimSpace(l))
			if upper != "DEBUG" && upper != "INFO" && upper != "WARN" && upper != "ERROR" {
				return nil, fmt.Errorf("invalid log level: %s", l)
			}
			levels = append(levels, upper)
		}
	}

	search := query.Get("search")

	var startTime, endTime *time.Time

	if start := query.Get("start"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return nil, fmt.Errorf("invalid start time format (use RFC3339): %s", start)
		}
		startTime = &t
	}

	if end := query.Get("end"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return nil, fmt.Errorf("invalid end time format (use RFC3339): %s", end)
		}
		endTime = &t
	}

	return logging.NewFilter(levels, search, startTime, endTime), nil
}

func shouldReadHistorical(filter *logging.Filter) bool {
	return filter.StartTime != nil || filter.EndTime == nil
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
	return nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
