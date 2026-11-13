//go:build unix

package server

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestNotifySignalsDeliversUntilStopped(t *testing.T) {
	guardSig := make(chan os.Signal, 1)
	signal.Notify(guardSig, syscall.SIGUSR1)
	defer signal.Stop(guardSig)

	sNoSig := New(withSignals())
	sigCh, stop := sNoSig.notifySignals()
	if sigCh == nil {
		t.Fatal("expected non-nil sigCh when signals is empty")
	}
	stop()

	sWithSig := New(withSignals(syscall.SIGUSR1))
	sigCh2, stop2 := sWithSig.notifySignals()
	if sigCh2 == nil {
		t.Fatal("expected non-nil sigCh when signals is non-empty")
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-sigCh2:
	case <-time.After(time.Second):
		t.Fatal("expected SIGUSR1 on sigCh2")
	}
	stop2()
	if err := syscall.Kill(os.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-guardSig:
	case <-time.After(time.Second):
		t.Fatal("expected SIGUSR1 on guardSig")
	}
	time.Sleep(10 * time.Millisecond)
	select {
	case <-sigCh2:
		t.Fatal("expected no SIGUSR1 on sigCh2 after stop2()")
	default:
	}
}
