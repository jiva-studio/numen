// Command fakesync serves the fake sync server on a local port for the end-to-end tests. It prints the
// address it listens on, then serves until it is stopped. With -heartbeat a stream of change hints sends a heartbeat after that silence. With -state-dir it keeps the data of the server in
// a file there and reads it again at start, so the process can stop and run again. With -subscriptions a new account has no
// subscription and its writes are refused until it pays at the address CreateCheckout returns. The admin paths /admin/backup,
// /admin/restore, /admin/announce and /admin/retention keep the state, put it back, announce the restore and delete the
// vaults, as the operator of a real server does; a backup and an announced epoch live in the memory of the process.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

const (
	stateFile    = "server.json"
	bodiesFolder = "bodies"
	payFolder    = "/pay/"
)

func newKeeper(fake *fakeserver.Server, path string, stopped *atomic.Bool, exit chan struct{}) *keeper {
	return &keeper{next: fake, fake: fake, path: path, mu: sync.Mutex{}, stopped: stopped, exit: exit, once: sync.Once{}}
}

// keeper saves the data of the server after every request that may have changed it: any but a GET, and the
// GET of a pay page. It writes the file only when the data differs from what it wrote last, so a request
// that read only leaves the file alone. The bodies of blobs are not in the file; the server writes each one to the bodies folder once.
type keeper struct {
	next    http.Handler
	fake    *fakeserver.Server
	path    string
	mu      sync.Mutex
	stopped *atomic.Bool
	exit    chan struct{}
	once    sync.Once
	saved   [sha256.Size]byte
}

func (k *keeper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer k.exitWhenStopped()
	k.next.ServeHTTP(w, r)
	if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, payFolder) {
		return
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.save(); err != nil {
		log.Print(err)
	}
}

func (k *keeper) exitWhenStopped() {
	if k.stopped.Load() {
		k.once.Do(func() { close(k.exit) })
	}
}

func (k *keeper) save() error {
	if k.path == "" {
		return nil
	}
	raw, err := k.fake.Snapshot()
	if err != nil {
		return err
	}
	if sum := sha256.Sum256(raw); sum == k.saved {
		return nil
	}
	if err := writeAtomic(k.path, raw); err != nil {
		return err
	}
	k.saved = sha256.Sum256(raw)
	return nil
}

func writeAtomic(path string, raw []byte) error {
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func getStatePath(fake *fakeserver.Server, stateDir string) string {
	if stateDir == "" {
		return ""
	}
	if err := prepareStateDir(stateDir); err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(stateDir, stateFile)
	if err := load(fake, path); err != nil {
		log.Fatal(err)
	}
	return path
}

func load(fake *fakeserver.Server, path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	head := make([]byte, len(fakeserver.SnapshotPrefix))
	if count, _ := io.ReadFull(file, head); string(head[:count]) != fakeserver.SnapshotPrefix {
		log.Printf("%s has the old layout and is not read; starting with an empty server", path)
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return fake.Restore(raw)
}

func prepareStateDir(dir string) error { return os.MkdirAll(dir, 0o700) }

func main() {
	address := flag.String("addr", "127.0.0.1:0", "address to listen on")
	protocol := flag.Uint64("protocol", 0, "protocol version the server needs; 0 is the version of this build")
	ttl := flag.Duration("session-ttl", time.Hour, "how long a session lives")
	subscriptions := flag.Bool("subscriptions", false, "make a new account have no subscription, so it must pay before it writes")
	plansFlag := flag.String("plans", "", "plans the server sells as id:GB pairs, such as basic:10,plus:100; empty sells basic, plus and pro")
	heartbeat := flag.Duration("heartbeat", syncprotocol.WatchHeartbeat, "silence after which a stream of change hints sends a heartbeat")
	stateDir := flag.String("state-dir", "", "folder that keeps the data of the server between runs; empty keeps it in memory")
	flag.Parse()
	if *protocol > math.MaxUint32 {
		log.Fatalf("-protocol %d does not fit in 32 bits", *protocol)
	}
	plans, err := readPlans(*plansFlag)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	var stopped atomic.Bool
	config := fakeserver.Config{
		Protocol: uint32(*protocol), Audience: url, SessionTTL: *ttl, NeedsSubscription: *subscriptions, Plans: plans,
		Heartbeat: *heartbeat, Stop: func() { stopped.Store(true) },
	}
	if *stateDir != "" {
		config.BodyDir = filepath.Join(*stateDir, bodiesFolder)
	}
	fake := fakeserver.New(config)
	exit := make(chan struct{})
	handler := newKeeper(fake, getStatePath(fake, *stateDir), &stopped, exit)
	if _, err := fmt.Fprintln(os.Stdout, url); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-exit
	if err := server.Close(); err != nil {
		log.Print(err)
	}
}
