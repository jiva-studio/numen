package portproxy

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// serveEcho starts a loopback TCP server that writes back what it reads and returns its address.
func serveEcho(t *testing.T) string {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return ln.Addr().String()
}

// openProxy opens a proxy on a free loopback port and returns it with its address.
func openProxy(t *testing.T, target Target, onError func(error)) (*Proxy, string) {
	t.Helper()
	p, err := Open(t.Context(), "127.0.0.1:0", target, onError)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return p, p.listener.Addr().String()
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestAConnectionIsForwardedToTheTargetAndReleasedWhenItCloses(t *testing.T) {
	echo := serveEcho(t)
	released := make(chan struct{})
	var opened atomic.Int32
	_, addr := openProxy(t, func(context.Context) (string, func(), error) {
		opened.Add(1)
		return echo, func() { close(released) }, nil
	}, func(err error) { t.Errorf("onError: %v", err) })

	conn := dial(t, addr)
	if _, err := io.WriteString(conn, "ping\n"); err != nil {
		t.Fatal(err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || reply != "ping\n" {
		t.Fatalf("reply = %q, %v, want the echo of the line", reply, err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Fatal("the target was not released after the connection closed")
	}
	if got := opened.Load(); got != 1 {
		t.Errorf("target opened %d times, want 1", got)
	}
}

func TestATargetErrorClosesTheConnectionAndIsReported(t *testing.T) {
	targetErr := errors.New("plugin disabled")
	reported := make(chan error, 1)
	_, addr := openProxy(t, func(context.Context) (string, func(), error) {
		return "", nil, targetErr
	}, func(err error) { reported <- err })

	conn := dial(t, addr)
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("the connection stayed open after the target failed")
	}
	select {
	case err := <-reported:
		if !errors.Is(err, targetErr) {
			t.Errorf("reported %v, want the target error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the target error was not reported")
	}
}

func TestCloseClosesTheListeningPortAndTheForwardedConnections(t *testing.T) {
	echo := serveEcho(t)
	released := make(chan struct{})
	p, addr := openProxy(t, func(context.Context) (string, func(), error) {
		return echo, func() { close(released) }, nil
	}, func(error) {})
	conn := dial(t, addr)
	if _, err := io.WriteString(conn, "ping\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := bufio.NewReader(conn).ReadString('\n'); err != nil {
		t.Fatal(err)
	}

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("the forwarded connection is open after Close")
	}
	select {
	case <-released:
	default:
		t.Error("Close returned before the target was released")
	}
	if second, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		_ = second.Close()
		t.Error("the port accepts connections after Close")
	}
	if err := p.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestCloseCancelsATargetThatIsOpening(t *testing.T) {
	isOpening := make(chan struct{})
	p, addr := openProxy(t, func(ctx context.Context) (string, func(), error) {
		close(isOpening)
		<-ctx.Done()
		return "", nil, ctx.Err()
	}, func(err error) { t.Errorf("onError: %v", err) })
	dial(t, addr)
	<-isOpening

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenFailsOnAPortThatIsTaken(t *testing.T) {
	taken, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = taken.Close() })

	p, err := Open(t.Context(), taken.Addr().String(), func(context.Context) (string, func(), error) {
		return "", nil, errors.New("unused")
	}, func(error) {})
	if err == nil {
		_ = p.Close()
		t.Fatal("Open succeeded on a port that is taken")
	}
}
