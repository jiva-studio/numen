// Package logfiles writes structured log entries to daily newline-delimited JSON files and removes
// daily files older than the retention window.
package logfiles

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

const (
	filePrefix = "numen-"
	fileSuffix = ".jsonl"
	dateLayout = "2006-01-02"

	// RetentionDays is how many days of daily log files are kept on disk.
	RetentionDays = 30

	// MaxDailyFileBytes bounds one daily log file so a runaway loop cannot exhaust disk space.
	MaxDailyFileBytes = 50 << 20

	privateDirMode  fs.FileMode = 0o700
	privateFileMode fs.FileMode = 0o600
)

// recordLine is the JSONL representation of one structured log entry on disk.
type recordLine struct {
	Time       string                     `json:"time"`
	Level      string                     `json:"level"`
	Source     string                     `json:"source"`
	Message    string                     `json:"message"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
}

// Files appends structured log entries to daily JSONL files inside a directory and prunes files
// older than RetentionDays. An empty directory path discards writes and reads nothing.
type Files struct {
	dir    string
	mu     sync.Mutex
	pruned string
}

// NewFiles returns the log file store for dir.
func NewFiles(dir string) *Files {
	return &Files{dir: dir}
}

// NewBesideIndex returns the log file store in a logs folder beside indexPath, or a discarding store when
// indexPath is empty.
func NewBesideIndex(indexPath string) *Files {
	if indexPath == "" {
		return NewFiles("")
	}
	return NewFiles(filepath.Join(filepath.Dir(indexPath), "logs"))
}

// WriteEntry appends one entry to the daily JSONL file for entry.Time.UTC() and prunes expired
// daily files once per day.
func (f *Files) WriteEntry(entry logs.Entry) error {
	if f == nil || f.dir == "" {
		return nil
	}
	day := entry.Time.UTC()
	dayKey := day.Format(dateLayout)
	line, err := encodeLine(entry)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.MkdirAll(f.dir, privateDirMode); err != nil {
		return fmt.Errorf("create log directory %s: %w", f.dir, err)
	}
	if f.pruned != dayKey {
		f.pruneOlderThanLocked(day.AddDate(0, 0, -RetentionDays))
		f.pruned = dayKey
	}

	target := filepath.Join(f.dir, filePrefix+dayKey+fileSuffix)
	if info, statErr := os.Stat(target); statErr == nil && info.Size()+int64(len(line)) > MaxDailyFileBytes {
		return nil
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, privateFileMode)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", target, err)
	}
	_, writeErr := file.Write(line)
	closeErr := file.Close()
	return errors.Join(writeErr, closeErr)
}

// ReadRecent reads up to limit most recent entries from the daily JSONL file for day.UTC().
func (f *Files) ReadRecent(day time.Time, limit int) ([]logs.Entry, error) {
	if f == nil || f.dir == "" || limit <= 0 {
		return nil, nil
	}
	target := filepath.Join(f.dir, filePrefix+day.UTC().Format(dateLayout)+fileSuffix)

	f.mu.Lock()
	raw, err := os.ReadFile(target)
	f.mu.Unlock()
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read log file %s: %w", target, err)
	}
	return decodeRecentLines(raw, limit), nil
}

func (f *Files) pruneOlderThanLocked(cutoff time.Time) {
	cutoffDay := time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, time.UTC)
	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return
	}
	for _, one := range entries {
		if one.IsDir() {
			continue
		}
		fileDay, ok := parseFileNameDate(one.Name())
		if !ok || !fileDay.Before(cutoffDay) {
			continue
		}
		if removeErr := os.Remove(filepath.Join(f.dir, one.Name())); removeErr != nil {
			continue
		}
	}
}

func parseFileNameDate(name string) (time.Time, bool) {
	datePart, hasPrefix := strings.CutPrefix(name, filePrefix)
	if !hasPrefix {
		return time.Time{}, false
	}
	datePart, hasSuffix := strings.CutSuffix(datePart, fileSuffix)
	if !hasSuffix {
		return time.Time{}, false
	}
	parsed, err := time.Parse(dateLayout, datePart)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func encodeLine(entry logs.Entry) ([]byte, error) {
	var props map[string]json.RawMessage
	if len(entry.Properties) > 0 {
		props = make(map[string]json.RawMessage, len(entry.Properties))
		for k, v := range entry.Properties {
			if json.Valid([]byte(v)) {
				props[k] = json.RawMessage(v)
			} else {
				quoted, err := json.Marshal(v)
				if err != nil {
					continue
				}
				props[k] = quoted
			}
		}
	}
	rec := recordLine{
		Time:       entry.Time.UTC().Format(time.RFC3339Nano),
		Level:      entry.Level.String(),
		Source:     entry.Source,
		Message:    entry.Message,
		Properties: props,
	}
	encoded, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func decodeRecentLines(raw []byte, limit int) []logs.Entry {
	var all []logs.Entry
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec recordLine
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		parsedTime, err := time.Parse(time.RFC3339Nano, rec.Time)
		if err != nil {
			continue
		}
		var props map[string]string
		if len(rec.Properties) > 0 {
			props = make(map[string]string, len(rec.Properties))
			for k, v := range rec.Properties {
				props[k] = string(v)
			}
		}
		all = append(all, logs.Entry{
			Time:       parsedTime,
			Level:      logs.ResolveLevel(rec.Level),
			Source:     rec.Source,
			Message:    rec.Message,
			Properties: props,
		})
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	return all
}
