package supervision

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"sync"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

const (
	// SourcePlugin marks a log entry that is a line a plugin process wrote to its output.
	SourcePlugin = "plugin"

	maxOutputLineBytes = 8 * 1024
	outputQueueLines   = 256
)

func forwardOutput(out io.Reader, record func(domainlogs.Entry), pluginID string, level domainlogs.Level) {
	if record == nil {
		discardOutput(out)
		return
	}
	queue := make(chan string, outputQueueLines)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for line := range queue {
			record(domainlogs.Entry{
				Level: level, Source: SourcePlugin, Message: line,
				Properties: map[string]string{"plugin": strconv.Quote(pluginID)},
			})
		}
	}()
	readLines(out, func(line string) {
		select {
		case queue <- line:
		default:
		}
	})
	close(queue)
	<-done
}

func readLines(in io.Reader, emit func(string)) {
	reader := bufio.NewReaderSize(in, maxOutputLineBytes)
	for {
		chunk, prefix, err := reader.ReadLine()
		if len(chunk) > 0 {
			if line := cutLine(chunk, prefix); line != "" {
				emit(line)
			}
			for prefix && err == nil {
				_, prefix, err = reader.ReadLine()
			}
		}
		if err != nil {
			return
		}
	}
}

func cutLine(chunk []byte, truncated bool) string {
	line := strings.TrimSpace(strings.ToValidUTF8(string(chunk), "�"))
	if truncated && line != "" {
		line = strings.TrimRight(line, "�") + "... [truncated]"
	}
	return line
}

func discardOutput(out io.Reader) {
	if _, err := io.Copy(io.Discard, out); err != nil {
		return
	}
}

// OutputLog holds where the lines a plugin process writes to its output are recorded.
type OutputLog struct {
	outputMu sync.RWMutex
	record   func(domainlogs.Entry)
}

// Set sets where the lines a plugin process writes to its output are recorded. Without it they are dropped.
func (l *OutputLog) Set(record func(domainlogs.Entry)) {
	l.outputMu.Lock()
	defer l.outputMu.Unlock()
	l.record = record
}

func (l *OutputLog) get() func(domainlogs.Entry) {
	l.outputMu.RLock()
	defer l.outputMu.RUnlock()
	return l.record
}
