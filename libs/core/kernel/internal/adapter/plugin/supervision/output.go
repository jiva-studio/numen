package supervision

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

const (
	// SourcePlugin marks a log entry that is a line a plugin process wrote to its output.
	SourcePlugin = "plugin"

	maxOutputLineBytes = 8 * 1024
	outputQueueLines   = 256
)

// pumpOutput records each line read from out as a log entry of the plugin until out ends. The lines pass
// through a bounded queue, so a slow log never blocks the plugin: when the queue is full the line is dropped.
func pumpOutput(out io.Reader, record func(domainlogs.Entry), pluginID string, level domainlogs.Level) {
	if record == nil {
		_, _ = io.Copy(io.Discard, out)
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

// readLines calls emit with every non-empty line of in, cut to maxOutputLineBytes; the rest of a longer line is skipped.
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
