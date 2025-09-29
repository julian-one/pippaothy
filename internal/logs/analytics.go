package logs

import (
	"strings"
)

// LogStats represents statistical data about logs
type LogStats struct {
	TotalRequests      int                    `json:"total_requests"`
	UniqueIPs          int                    `json:"unique_ips"`
	ErrorCount         int                    `json:"error_count"`
	WarningCount       int                    `json:"warning_count"`
	AvgResponseTime    float64                `json:"avg_response_time"`
	StatusCodeDist     map[string]int         `json:"status_code_dist"`
	TopPaths           []PathStat             `json:"top_paths"`
	RequestsPerHour    []TimeSeriesPoint      `json:"requests_per_hour"`
	ErrorsPerHour      []TimeSeriesPoint      `json:"errors_per_hour"`
	MethodDistribution map[string]int         `json:"method_distribution"`
	UserAgentStats     []UserAgentStat        `json:"user_agent_stats"`
	LogTypeBreakdown   map[string]int         `json:"log_type_breakdown"`
}

// PathStat represents statistics for a specific path
type PathStat struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// TimeSeriesPoint represents a point in time series data
type TimeSeriesPoint struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// UserAgentStat represents user agent statistics
type UserAgentStat struct {
	Browser string `json:"browser"`
	Count   int    `json:"count"`
}

// AnalyzeLogs analyzes log entries and returns statistics
func AnalyzeLogs(entries []LogEntry) LogStats {
	stats := LogStats{
		StatusCodeDist:     make(map[string]int),
		MethodDistribution: make(map[string]int),
		LogTypeBreakdown:   make(map[string]int),
	}
	
	ipSet := make(map[string]bool)
	pathCounts := make(map[string]int)
	hourlyRequests := make(map[string]int)
	hourlyErrors := make(map[string]int)
	userAgentCounts := make(map[string]int)
	
	for _, entry := range entries {
		stats.TotalRequests++
		
		// Determine log type (nginx vs application)
		logType := "nginx"
		if strings.Contains(entry.Message, "HTTP request") || 
		   strings.Contains(entry.Message, "user authenticated") ||
		   strings.Contains(entry.Message, "session") {
			logType = "application"
		}
		stats.LogTypeBreakdown[logType]++
		
		// Count unique IPs
		if entry.ClientIP != "" {
			ipSet[entry.ClientIP] = true
		}
		
		// Count errors and warnings
		switch entry.Level {
		case "error":
			stats.ErrorCount++
		case "warn":
			stats.WarningCount++
		}
		
		// Extract and count status codes from nginx logs
		statusCode := extractStatusCode(entry.Message)
		if statusCode != "" {
			stats.StatusCodeDist[statusCode]++
		}
		
		// Count paths
		if entry.Path != "" {
			// Skip static assets for top paths
			if !strings.HasPrefix(entry.Path, "/static/") {
				pathCounts[entry.Path]++
			}
		}
		
		// Count methods
		if entry.Method != "" {
			stats.MethodDistribution[entry.Method]++
		}
		
		// Group by hour
		hour := entry.Timestamp.Format("2006-01-02 15h")
		hourlyRequests[hour]++
		if entry.Level == "error" {
			hourlyErrors[hour]++
		}
		
		// Parse user agent
		userAgent := extractUserAgent(entry.Message)
		if userAgent != "" {
			browser := parseUserAgent(userAgent)
			userAgentCounts[browser]++
		}
	}
	
	stats.UniqueIPs = len(ipSet)
	
	// Convert path counts to sorted top paths
	stats.TopPaths = getTopPaths(pathCounts, 10)
	
	// Convert hourly data to time series
	stats.RequestsPerHour = convertToTimeSeries(hourlyRequests)
	stats.ErrorsPerHour = convertToTimeSeries(hourlyErrors)
	
	// Convert user agent data
	stats.UserAgentStats = convertUserAgentStats(userAgentCounts)
	
	return stats
}

// extractStatusCode extracts HTTP status code from log message
func extractStatusCode(message string) string {
	// Look for pattern like " 200 " or " 404 "
	parts := strings.Fields(message)
	for i, part := range parts {
		if len(part) == 3 && isNumeric(part) {
			// Check if previous part is HTTP/2.0 or similar
			if i > 0 && strings.Contains(parts[i-1], "HTTP") {
				return part
			}
		}
	}
	return ""
}

// extractUserAgent extracts user agent from nginx log
func extractUserAgent(message string) string {
	// User agent is typically the last quoted string in nginx logs
	lastQuote := strings.LastIndex(message, "\"")
	if lastQuote == -1 {
		return ""
	}
	secondLastQuote := strings.LastIndex(message[:lastQuote], "\"")
	if secondLastQuote == -1 {
		return ""
	}
	return message[secondLastQuote+1 : lastQuote]
}

// parseUserAgent converts user agent string to browser name
func parseUserAgent(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		return "Safari"
	case strings.Contains(ua, "chrome"):
		return "Chrome"
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "edge"):
		return "Edge"
	case strings.Contains(ua, "bot") || strings.Contains(ua, "crawler"):
		return "Bot"
	default:
		return "Other"
	}
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// getTopPaths returns the top N paths by count
func getTopPaths(pathCounts map[string]int, limit int) []PathStat {
	var paths []PathStat
	for path, count := range pathCounts {
		paths = append(paths, PathStat{Path: path, Count: count})
	}
	
	// Sort by count (descending)
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			if paths[j].Count > paths[i].Count {
				paths[i], paths[j] = paths[j], paths[i]
			}
		}
	}
	
	if len(paths) > limit {
		paths = paths[:limit]
	}
	
	return paths
}

// convertToTimeSeries converts hourly data to time series points
func convertToTimeSeries(hourlyData map[string]int) []TimeSeriesPoint {
	var points []TimeSeriesPoint
	for time, count := range hourlyData {
		points = append(points, TimeSeriesPoint{Time: time, Count: count})
	}
	
	// Sort by time
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[j].Time < points[i].Time {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
	
	// Limit to last 24 hours
	if len(points) > 24 {
		points = points[len(points)-24:]
	}
	
	return points
}

// convertUserAgentStats converts user agent counts to stats
func convertUserAgentStats(userAgentCounts map[string]int) []UserAgentStat {
	var stats []UserAgentStat
	for browser, count := range userAgentCounts {
		stats = append(stats, UserAgentStat{Browser: browser, Count: count})
	}
	
	// Sort by count (descending)
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	
	return stats
}

// GetLogsDashboard returns logs with analytics for dashboard
func GetLogsDashboard() (LogResult, LogStats) {
	query := LogQuery{
		Page:  1,
		Limit: 1000, // Get more logs for analytics
	}
	
	result := GetLogs(query)
	stats := AnalyzeLogs(result.Entries)
	
	// Return only recent entries for display
	if len(result.Entries) > 50 {
		result.Entries = result.Entries[:50]
		result.Limit = 50
	}
	
	return result, stats
}