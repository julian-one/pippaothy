package logging

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Reader reads historical logs from rotated files.
type Reader struct {
	basePath string
}

// NewReader creates a new log reader.
func NewReader(basePath string) *Reader {
	return &Reader{basePath: basePath}
}

// ReadHistorical reads log entries from historical files matching the filter.
// Files are read in chronological order (oldest first).
func (r *Reader) ReadHistorical(ctx context.Context, filter *Filter, out chan<- *LogEntry) error {
	defer close(out)

	files, err := r.getLogFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := r.readFile(ctx, file, filter, out); err != nil {
			continue
		}
	}

	return nil
}

// getLogFiles returns all log files sorted by modification time (oldest first).
func (r *Reader) getLogFiles() ([]string, error) {
	dir := filepath.Dir(r.basePath)
	base := filepath.Base(r.basePath)
	baseWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, baseWithoutExt) {
			files = append(files, filepath.Join(dir, name))
		}
	}

	sort.Slice(files, func(i, j int) bool {
		infoI, errI := os.Stat(files[i])
		infoJ, errJ := os.Stat(files[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	return files, nil
}

// readFile reads a single log file (handles .gz compression).
func (r *Reader) readFile(ctx context.Context, path string, filter *Filter, out chan<- *LogEntry) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file

	if strings.HasSuffix(path, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if filter == nil || filter.Matches(&entry) {
			out <- &entry
		}
	}

	return scanner.Err()
}
