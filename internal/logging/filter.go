package logging

import (
	"strings"
	"time"
)

// Filter defines criteria for log filtering.
type Filter struct {
	Levels    []string
	Search    string
	StartTime *time.Time
	EndTime   *time.Time
}

// NewFilter creates a filter from parameters.
func NewFilter(levels []string, search string, start, end *time.Time) *Filter {
	return &Filter{
		Levels:    levels,
		Search:    search,
		StartTime: start,
		EndTime:   end,
	}
}

// Matches checks if a log entry matches the filter criteria.
func (f *Filter) Matches(entry *LogEntry) bool {
	if len(f.Levels) > 0 {
		levelMatch := false
		for _, l := range f.Levels {
			if strings.EqualFold(entry.Level, l) {
				levelMatch = true
				break
			}
		}
		if !levelMatch {
			return false
		}
	}

	if f.Search != "" {
		if !strings.Contains(
			strings.ToLower(entry.Message),
			strings.ToLower(f.Search),
		) {
			return false
		}
	}

	if f.StartTime != nil && entry.Time.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && entry.Time.After(*f.EndTime) {
		return false
	}

	return true
}

// IsEmpty returns true if no filter criteria are set.
func (f *Filter) IsEmpty() bool {
	return len(f.Levels) == 0 &&
		f.Search == "" &&
		f.StartTime == nil &&
		f.EndTime == nil
}
