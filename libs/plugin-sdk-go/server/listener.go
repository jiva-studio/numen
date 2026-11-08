package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// listenerPort is the name and the bound port of a listener opened by OpenListener.
type listenerPort struct {
	name string
	port int
}

// OpenListener binds 127.0.0.1:0 for the listener the manifest declares under name; the ready line
// reports its port. Called from mount, before Serve. The server closes the listener when Serve
// returns, or on Close.
func (s *Server) OpenListener(name string) (net.Listener, error) {
	for _, entry := range s.listeners {
		if entry.name == name {
			return nil, fmt.Errorf("open listener %s: %w", name, errDuplicateListener)
		}
	}
	ln, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("open listener %s: %w", name, err)
	}
	addr, isTCP := ln.Addr().(*net.TCPAddr)
	if !isTCP {
		return nil, errors.Join(fmt.Errorf("listener %s address %s is not a TCP address", name, ln.Addr()), ln.Close())
	}
	s.listeners = append(s.listeners, listenerPort{name: name, port: addr.Port})
	s.OnClose(func() error {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			return fmt.Errorf("close listener %s: %w", name, err)
		}
		return nil
	})
	return ln, nil
}

var errDuplicateListener = errors.New("the listener is already open")

func (s *Server) formatReadyLine() string {
	var line strings.Builder
	line.WriteString("NUMEN_READY port=" + strconv.Itoa(s.actualPort))
	for _, entry := range s.listeners {
		line.WriteString(" listener." + entry.name + "=" + strconv.Itoa(entry.port))
	}
	line.WriteString("\n")
	return line.String()
}
