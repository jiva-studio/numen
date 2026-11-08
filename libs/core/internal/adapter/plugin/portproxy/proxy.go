// Package portproxy binds a loopback port and forwards each TCP connection it accepts to a target
// that is opened for the connection.
package portproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const (
	minAcceptRetryDelay = 5 * time.Millisecond
	maxAcceptRetryDelay = time.Second
)

// ErrTargetReported marks a target error whose cause was reported before. The proxy closes the
// connection and does not pass the error to onError.
var ErrTargetReported = errors.New("the target error was reported before")

// Target opens the connection a proxied connection is forwarded to, and keeps the plugin active for it
// until release is called.
type Target func(ctx context.Context) (addr string, release func(), err error)

// Proxy is a bound listening port that forwards each accepted TCP connection to Target.
type Proxy struct {
	listener net.Listener
	target   Target
	onError  func(error)
	cancel   context.CancelFunc

	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	isClosed bool
	wg       sync.WaitGroup
}

// Open binds addr and forwards each connection accepted on it to the address target returns.
// onError receives the failures of a connection; the port stays bound after them.
func Open(ctx context.Context, addr string, target Target, onError func(error)) (*Proxy, error) {
	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", addr, err)
	}
	return newProxy(ctx, listener, target, onError), nil
}

func newProxy(ctx context.Context, listener net.Listener, target Target, onError func(error)) *Proxy {
	proxyCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p := &Proxy{
		listener: listener,
		target:   target,
		onError:  onError,
		cancel:   cancel,
		mu:       sync.Mutex{},
		conns:    make(map[net.Conn]struct{}),
		isClosed: false,
		wg:       sync.WaitGroup{},
	}
	p.wg.Add(1)
	go p.acceptConnections(proxyCtx)
	return p
}

// Close closes the listening socket and every forwarded connection, and waits until the
// forwarding of each connection has ended.
func (p *Proxy) Close() error {
	p.mu.Lock()
	if p.isClosed {
		p.mu.Unlock()
		return nil
	}
	p.isClosed = true
	conns := make([]net.Conn, 0, len(p.conns))
	for conn := range p.conns {
		conns = append(conns, conn)
	}
	p.mu.Unlock()

	p.cancel()
	err := p.listener.Close()
	if errors.Is(err, net.ErrClosed) {
		err = nil
	}
	for _, conn := range conns {
		p.closeConn(conn)
	}
	p.wg.Wait()
	if err != nil {
		return fmt.Errorf("close listener on %s: %w", p.listener.Addr(), err)
	}
	return nil
}

func (p *Proxy) acceptConnections(ctx context.Context) {
	defer p.wg.Done()
	var retryDelay time.Duration
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			p.onError(fmt.Errorf("accept on %s: %w", p.listener.Addr(), err))
			retryDelay = min(max(2*retryDelay, minAcceptRetryDelay), maxAcceptRetryDelay)
			if !waitFor(ctx, retryDelay) {
				return
			}
			continue
		}
		retryDelay = 0
		if !p.track(conn) {
			p.closeConn(conn)
			return
		}
		p.wg.Add(1)
		go p.forward(ctx, conn)
	}
}

func waitFor(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (p *Proxy) track(conn net.Conn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.isClosed {
		return false
	}
	p.conns[conn] = struct{}{}
	return true
}

func (p *Proxy) untrack(conn net.Conn) {
	p.mu.Lock()
	delete(p.conns, conn)
	p.mu.Unlock()
	p.closeConn(conn)
}

func (p *Proxy) forward(ctx context.Context, client net.Conn) {
	defer p.wg.Done()
	defer p.untrack(client)

	addr, release, err := p.target(ctx)
	if err != nil {
		if !errors.Is(err, ErrTargetReported) {
			p.reportUnlessClosed(ctx, fmt.Errorf("open the target of a connection to %s: %w", p.listener.Addr(), err))
		}
		return
	}
	defer release()

	upstream, err := new(net.Dialer).DialContext(ctx, "tcp", addr)
	if err != nil {
		p.reportUnlessClosed(ctx, fmt.Errorf("dial %s: %w", addr, err))
		return
	}
	if !p.track(upstream) {
		p.closeConn(upstream)
		return
	}
	defer p.untrack(upstream)

	var copies sync.WaitGroup
	copies.Add(2)
	go func() {
		defer copies.Done()
		p.copyStream(upstream, client)
	}()
	go func() {
		defer copies.Done()
		p.copyStream(client, upstream)
	}()
	copies.Wait()
}

func (p *Proxy) copyStream(dst, src net.Conn) {
	if _, err := io.Copy(dst, src); err != nil {
		p.closeConn(dst)
		p.closeConn(src)
		return
	}
	if tcp, isTCP := dst.(*net.TCPConn); isTCP {
		if err := tcp.CloseWrite(); err != nil && !errors.Is(err, net.ErrClosed) {
			p.closeConn(dst)
			p.closeConn(src)
		}
		return
	}
	p.closeConn(dst)
}

func (p *Proxy) closeConn(conn net.Conn) {
	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		p.onError(fmt.Errorf("close connection %s: %w", conn.RemoteAddr(), err))
	}
}

func (p *Proxy) reportUnlessClosed(ctx context.Context, err error) {
	if ctx.Err() == nil {
		p.onError(err)
	}
}
