package logs

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// LogEntry represents a single log entry with essential fields
type LogEntry struct {
	// Core fields
	Timestamp time.Time `json:"time"`
	Level     string    `json:"level"`
	Message   string    `json:"msg"`

	// Request information
	RequestID string `json:"request_id,omitempty"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	ClientIP  string `json:"client_ip,omitempty"`
}

// LogQuery represents query parameters for log filtering
type LogQuery struct {
	// Pagination
	Page  int `json:"page"`
	Limit int `json:"limit"`

	// Filtering
	Level string `json:"level,omitempty"`

	// Grouping - "date", "hour", "ip", or empty for no grouping
	GroupBy string `json:"group_by,omitempty"`
}

// LogResult represents the result of a log query
type LogResult struct {
	// Results
	Entries []LogEntry            `json:"entries,omitempty"`
	Groups  map[string][]LogEntry `json:"groups,omitempty"` // For grouped results

	// Pagination metadata
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`

	// Error information
	Error string `json:"error,omitempty"`
}

// Get log file path with fallback
func GetLogFilePath() string {
	if _, err := os.Stat("/mnt/ssd/logs/access.log"); err == nil {
		return "/mnt/ssd/logs/access.log"
	}
	return "./logs/access.log"
}

// Parse query parameters from HTTP request
func ParseQuery(r *http.Request) LogQuery {
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	level := r.URL.Query().Get("level")
	groupBy := r.URL.Query().Get("groupBy")

	return LogQuery{
		Page:    page,
		Limit:   limit,
		Level:   level,
		GroupBy: groupBy,
	}
}

// Parse a single log line - simplified to handle only JSON logs
func parseLogLine(line string) (LogEntry, bool) {
	var entry LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		// If not JSON, try to extract timestamp from line or use zero time
		timestamp := time.Time{}
		if len(line) > 25 {
			// Try to parse common timestamp formats at beginning of line
			if t, parseErr := time.Parse("2006-01-02T15:04:05", line[:19]); parseErr == nil {
				timestamp = t
			}
		}
		entry = LogEntry{
			Timestamp: timestamp,
			Level:     "info",
			Message:   line,
		}
	}
	return entry, true
}

// Check if entry matches filters
func matchesFilters(entry LogEntry, query LogQuery) bool {
	if query.Level != "" && entry.Level != query.Level {
		return false
	}
	return true
}

// Get recent log entries - simplified approach
func GetLogs(query LogQuery) LogResult {
	logFile := GetLogFilePath()

	// Check if file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return LogResult{
			Entries: []LogEntry{},
			Page:    query.Page,
			Limit:   query.Limit,
			HasMore: false,
			Error:   "Log file not found",
		}
	}

	file, err := os.Open(logFile)
	if err != nil {
		return LogResult{
			Entries: []LogEntry{},
			Page:    query.Page,
			Limit:   query.Limit,
			HasMore: false,
			Error:   "Failed to open log file",
		}
	}
	defer file.Close()

	// Read all matching entries (simplified - no streaming complexity)
	var allEntries []LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, ok := parseLogLine(line)
		if !ok {
			continue
		}

		if matchesFilters(entry, query) {
			allEntries = append(allEntries, entry)
		}
	}

	if err := scanner.Err(); err != nil {
		return LogResult{
			Entries: []LogEntry{},
			Page:    query.Page,
			Limit:   query.Limit,
			HasMore: false,
			Error:   "Failed to read log file",
		}
	}

	// Reverse to show newest first
	for i := len(allEntries)/2 - 1; i >= 0; i-- {
		opp := len(allEntries) - 1 - i
		allEntries[i], allEntries[opp] = allEntries[opp], allEntries[i]
	}

	// Handle grouping if requested
	if query.GroupBy != "" {
		groups := make(map[string][]LogEntry)

		for _, entry := range allEntries {
			var key string
			switch query.GroupBy {
			case "date":
				key = entry.Timestamp.Format("2006-01-02")
			case "hour":
				key = entry.Timestamp.Format("2006-01-02 15h")
			case "ip":
				if entry.ClientIP != "" {
					key = entry.ClientIP
				} else {
					key = "No IP"
				}
			default:
				key = "Unknown"
			}
			groups[key] = append(groups[key], entry)
		}

		return LogResult{
			Groups:  groups,
			Page:    query.Page,
			Limit:   query.Limit,
			HasMore: false,
		}
	}

	return paginate(allEntries, query.Page, query.Limit)
}

func paginate(entries []LogEntry, page, limit int) LogResult {
	total := len(entries)
	start := (page - 1) * limit

	if start >= total {
		return LogResult{
			Page:  page,
			Limit: limit,
		}
	}

	end := min(start+limit, total)

	return LogResult{
		Entries: entries[start:end],
		Page:    page,
		Limit:   limit,
		HasMore: end < total,
	}
}
