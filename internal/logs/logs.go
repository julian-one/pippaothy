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

// UnmarshalJSON custom unmarshaler to handle different timestamp formats
func (le *LogEntry) UnmarshalJSON(data []byte) error {
	type Alias LogEntry
	aux := &struct {
		TimeStr string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(le),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Parse timestamp with multiple format support
	if aux.TimeStr != "" {
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}

		var parsed bool
		for _, format := range formats {
			if t, err := time.Parse(format, aux.TimeStr); err == nil {
				le.Timestamp = t
				parsed = true
				break
			}
		}

		if !parsed {
			// If we can't parse the timestamp, use current time
			le.Timestamp = time.Now()
		}
	}

	return nil
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
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	HasMore    bool `json:"has_more"`
	TotalCount int  `json:"total_count"`
	TotalPages int  `json:"total_pages"`

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

// Parse a single log line - handles both JSON and nginx access logs
func parseLogLine(line string) (LogEntry, bool) {
	var entry LogEntry
	// First try to parse as JSON
	if err := json.Unmarshal([]byte(line), &entry); err == nil {
		return entry, true
	}

	// If not JSON, parse as nginx access log
	// Format: IP - - [timestamp] "method path protocol" status size "referer" "user-agent"
	timestamp := time.Now() // Default to now if parsing fails

	// Find the timestamp between square brackets
	startIdx := strings.Index(line, "[")
	endIdx := strings.Index(line, "]")
	if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
		timeStr := line[startIdx+1 : endIdx]
		// Parse nginx timestamp format: 14/Sep/2025:17:02:35 +0000
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeStr); err == nil {
			timestamp = t
		}
	}

	// Extract other fields from nginx log
	level := "info"

	// Check for status code to determine level
	if strings.Contains(line, " 404 ") || strings.Contains(line, " 403 ") {
		level = "warn"
	} else if strings.Contains(line, " 500 ") || strings.Contains(line, " 502 ") || strings.Contains(line, " 503 ") {
		level = "error"
	}

	// Extract request path
	requestStart := strings.Index(line, "\"")
	requestEnd := strings.Index(line[requestStart+1:], "\"")
	path := ""
	method := ""
	if requestStart != -1 && requestEnd != -1 {
		request := line[requestStart+1 : requestStart+1+requestEnd]
		parts := strings.Fields(request)
		if len(parts) >= 2 {
			method = parts[0]
			path = parts[1]
		}
	}

	// Extract client IP
	clientIP := ""
	if idx := strings.Index(line, " "); idx > 0 {
		clientIP = line[:idx]
	}

	entry = LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Message:   line,
		Method:    method,
		Path:      path,
		ClientIP:  clientIP,
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
			Entries:    []LogEntry{},
			Page:       query.Page,
			Limit:      query.Limit,
			HasMore:    false,
			TotalCount: 0,
			TotalPages: 0,
			Error:      "Log file not found",
		}
	}

	file, err := os.Open(logFile)
	if err != nil {
		return LogResult{
			Entries:    []LogEntry{},
			Page:       query.Page,
			Limit:      query.Limit,
			HasMore:    false,
			TotalCount: 0,
			TotalPages: 0,
			Error:      "Failed to open log file",
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
			Entries:    []LogEntry{},
			Page:       query.Page,
			Limit:      query.Limit,
			HasMore:    false,
			TotalCount: 0,
			TotalPages: 0,
			Error:      "Failed to read log file",
		}
	}

	// Reverse to show newest first
	for i := len(allEntries)/2 - 1; i >= 0; i-- {
		opp := len(allEntries) - 1 - i
		allEntries[i], allEntries[opp] = allEntries[opp], allEntries[i]
	}

	totalCount := len(allEntries)
	totalPages := (totalCount + query.Limit - 1) / query.Limit
	if totalPages == 0 {
		totalPages = 1
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
			Groups:     groups,
			Page:       query.Page,
			Limit:      query.Limit,
			HasMore:    false,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}
	}

	return paginate(allEntries, query.Page, query.Limit, totalCount, totalPages)
}

func paginate(entries []LogEntry, page, limit, totalCount, totalPages int) LogResult {
	total := len(entries)
	start := (page - 1) * limit

	if start >= total {
		return LogResult{
			Page:       page,
			Limit:      limit,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}
	}

	end := min(start+limit, total)

	return LogResult{
		Entries:    entries[start:end],
		Page:       page,
		Limit:      limit,
		HasMore:    end < total,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}
}
