package portproxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func serveEcho(t *testing.T) string {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var echoes sync.WaitGroup
	t.Cleanup(func() {
		testsupport.Close(t, ln)
		echoes.Wait()
	})
	echoes.Go(func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			echoes.Go(func() {
				defer testsupport.Close(t, conn)
				if _, err := io.Copy(conn, conn); err != nil {
					t.Logf("echo: %v", err)
				}
			})
		}
	})
	return ln.Addr().String()
}

func openProxy(t *testing.T, target Target, onError func(error)) (*Proxy, string) {
	t.Helper()
	p, err := Open(t.Context(), "127.0.0.1:0", target, onError)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, p)
	return p, p.listener.Addr().String()
}

func dialLoopback(ctx context.Context, addr string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: time.Second}
	return dialer.DialContext(ctx, "tcp", addr)
}

func dial(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := dialLoopback(t.Context(), addr)
	if err != nil {
		t.Fatal(err)
	}
	closeConn := func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close: %v", err)
		}
	}
	// The connection is closed after 5 seconds, so a read that gets no reply ends the test.
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	stop := context.AfterFunc(ctx, closeConn)
	t.Cleanup(func() {
		stop()
		cancel()
		closeConn()
	})
	return conn
}

func TestAConnectionIsForwardedToTheTargetAndReleasedWhenItCloses(t *testing.T) {
	echo := serveEcho(t)
	released := make(chan struct{})
	var openCount atomic.Int32
	_, addr := openProxy(t, func(context.Context) (string, func(), error) {
		openCount.Add(1)
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
	if got := openCount.Load(); got != 1 {
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
	if second, err := dialLoopback(t.Context(), addr); err == nil {
		testsupport.Close(t, second)
		t.Error("the port accepts connections after Close")
	}
	if err := p.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestCloseCancelsATargetThatIsOpening(t *testing.T) {
	isOpenStarted := make(chan struct{})
	p, addr := openProxy(t, func(ctx context.Context) (string, func(), error) {
		close(isOpenStarted)
		<-ctx.Done()
		return "", nil, ctx.Err()
	}, func(err error) { t.Errorf("onError: %v", err) })
	dial(t, addr)
	<-isOpenStarted

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenFailsOnAPortThatIsTaken(t *testing.T) {
	taken, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, taken)

	p, err := Open(t.Context(), taken.Addr().String(), func(context.Context) (string, func(), error) {
		return "", nil, errors.New("unused")
	}, func(error) {})
	if err == nil {
		testsupport.Close(t, p)
		t.Fatal("Open succeeded on a port that is taken")
	}
}

// errorLog collects the errors a proxy reports.
type errorLog struct {
	mu   sync.Mutex
	errs []error
}

func (l *errorLog) add(err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, err)
}

func (l *errorLog) get() []error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.errs)
}

// failingListener returns an EMFILE accept error from the first failures calls of Accept, or from
// every call when failures is negative, and delegates to the loopback listener otherwise. After Close,
// Accept returns net.ErrClosed.
type failingListener struct {
	net.Listener
	mu       sync.Mutex
	failures int
	isClosed bool
}

func newFailingListener(t *testing.T, failures int) *failingListener {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return &failingListener{Listener: ln, failures: failures}
}

func (l *failingListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.isClosed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	if l.failures != 0 {
		if l.failures > 0 {
			l.failures--
		}
		l.mu.Unlock()
		return nil, &net.OpError{Op: "accept", Net: "tcp", Err: syscall.EMFILE}
	}
	l.mu.Unlock()
	return l.Listener.Accept()
}

func (l *failingListener) Close() error {
	l.mu.Lock()
	l.isClosed = true
	l.mu.Unlock()
	return l.Listener.Close()
}

func TestAnAcceptErrorIsReportedAndTheProxyKeepsAccepting(t *testing.T) {
	echo := serveEcho(t)
	ln := newFailingListener(t, 3)
	var reports errorLog
	p := newProxy(t.Context(), ln, func(context.Context) (string, func(), error) {
		return echo, func() {}, nil
	}, reports.add)
	testsupport.CloseAfter(t, p)

	conn := dial(t, ln.Addr().String())
	if _, err := io.WriteString(conn, "after the accept errors\n"); err != nil {
		t.Fatal(err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || reply != "after the accept errors\n" {
		t.Fatalf("reply = %q, %v, want the echo of the line", reply, err)
	}

	errs := reports.get()
	if len(errs) != 3 {
		t.Fatalf("reported %d errors (%v), want the 3 accept errors", len(errs), errs)
	}
	for _, err := range errs {
		if !errors.Is(err, syscall.EMFILE) {
			t.Errorf("reported %v, want the accept error", err)
		}
	}
}

func TestProxyBacksOffOnAcceptErrorsAndCloseEndsTheWait(t *testing.T) {
	ln := newFailingListener(t, -1)
	var reports errorLog
	// The buffer contains more reports than the proxy makes before Close.
	reported := make(chan struct{}, 64)
	p := newProxy(t.Context(), ln, func(context.Context) (string, func(), error) {
		return "", nil, errors.New("unused")
	}, func(err error) {
		reports.add(err)
		select {
		case reported <- struct{}{}:
		default:
		}
	})
	testsupport.CloseAfter(t, p)

	patience := time.NewTimer(testsupport.Patience)
	defer patience.Stop()
	select {
	case <-reported:
	case <-patience.C:
		t.Fatal("no accept error was reported")
	}
	// The waits after the first 8 errors are 5, 10, ... 640 ms, 1275 ms in all; the 9th error is
	// followed by a wait of 1 s.
	oneSecond := time.NewTimer(time.Second)
	defer oneSecond.Stop()
	isSecondOver := false
	for count := 1; count < 9; {
		select {
		case <-reported:
			count++
			if count == 9 && !isSecondOver {
				select {
				case <-oneSecond.C:
					isSecondOver = true
				default:
				}
			}
			if count == 9 && !isSecondOver {
				t.Error("9 accept errors were reported within 1s, want the waits between them to add up to over 1s")
			}
		case <-oneSecond.C:
			isSecondOver = true
		case <-patience.C:
			t.Fatalf("%d accept errors were reported, want 9", count)
		}
	}

	closed := make(chan error, 1)
	go func() { closed <- p.Close() }()
	closeLimit := time.NewTimer(300 * time.Millisecond)
	defer closeLimit.Stop()
	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("Close: %v", err)
		}
	case <-closeLimit.C:
		t.Fatal("Close did not return within 300ms while the proxy waited after an accept error")
	}
	for _, err := range reports.get() {
		if errors.Is(err, net.ErrClosed) {
			t.Errorf("reported %v, want the close of the listener not reported", err)
		}
	}
}

func TestErrTargetReportedClosesConnectionWithoutReport(t *testing.T) {
	reportedErr := fmt.Errorf("activate the plugin again: %w", ErrTargetReported)
	plainErr := errors.New("plugin disabled")
	targetErrs := []error{reportedErr, plainErr, plainErr}
	wantReports := []int{0, 1, 2}
	var openCount atomic.Int32
	var reports errorLog
	_, addr := openProxy(t, func(context.Context) (string, func(), error) {
		i := int(openCount.Add(1)) - 1
		if i >= len(targetErrs) {
			return "", nil, plainErr
		}
		return "", nil, targetErrs[i]
	}, reports.add)

	for i := range targetErrs {
		conn := dial(t, addr)
		if _, err := conn.Read(make([]byte, 1)); err == nil {
			t.Fatalf("connection %d stayed open after the target failed", i+1)
		}
		testsupport.Close(t, conn)

		errs := reports.get()
		if len(errs) != wantReports[i] {
			t.Fatalf("after connection %d: reported %d errors (%v), want %d", i+1, len(errs), errs, wantReports[i])
		}
		for _, err := range errs {
			if !errors.Is(err, plainErr) {
				t.Errorf("reported %v, want only the target error that does not wrap ErrTargetReported", err)
			}
		}
	}
}
