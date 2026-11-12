package server

import (
	"io"
	"os"
	"time"
)

// The options below set what New takes from the process: the tests serve on a port of their own, read
// the ready line and hold the standard input.

func withPort(port int) Option {
	return func(s *Server) { s.port = port }
}

func withToken(token string) Option {
	return func(s *Server) { s.token = token }
}

func withStdout(w io.Writer) Option {
	return func(s *Server) { s.stdout = w }
}

func withStdin(r io.Reader) Option {
	return func(s *Server) { s.stdin = r }
}

func withSignals(signals ...os.Signal) Option {
	return func(s *Server) { s.signals = signals }
}

func withShutdownTimeout(d time.Duration) Option {
	return func(s *Server) { s.shutdownTimeout = d }
}
