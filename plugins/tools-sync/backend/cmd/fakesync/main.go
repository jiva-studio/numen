// Command fakesync serves the fake sync server on a local port for the end-to-end tests. It prints the
// address it listens on, then serves until it is stopped. With -state-dir it keeps the data of the server in
// a file there and reads it again at start, so the process can stop and run again. With -subscriptions a new account has no
// subscription and its writes are refused until it pays at the address CreateCheckout returns.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

const (
	stateFile = "server.json"
	payFolder = "/pay/"
)

// keeper saves the data of the server after every request that may have changed it: any but a GET, and the
// GET of a pay page.
type keeper struct {
	next http.Handler
	fake *fakeserver.Server
	path string
	mu   sync.Mutex
}

func (k *keeper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (k *keeper) save() error {
	raw, err := k.fake.Snapshot()
	if err != nil {
		return err
	}
	temporary := k.path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, k.path)
}

func load(fake *fakeserver.Server, path string) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return fake.Restore(raw)
}

func main() {
	address := flag.String("addr", "127.0.0.1:0", "address to listen on")
	protocol := flag.Uint("protocol", 0, "protocol version the server needs; 0 is the version of this build")
	ttl := flag.Duration("session-ttl", time.Hour, "how long a session lives")
	subscriptions := flag.Bool("subscriptions", false, "make a new account have no subscription, so it must pay before it writes")
	plansFlag := flag.String("plans", "", "plans the server sells as id:GB pairs, such as basic:10,plus:100; empty sells basic, plus and pro")
	stateDir := flag.String("state-dir", "", "folder that keeps the data of the server between runs; empty keeps it in memory")
	flag.Parse()
	plans, err := readPlans(*plansFlag)
	if err != nil {
		log.Fatal(err)
	}

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	fake := fakeserver.New(fakeserver.Config{Protocol: uint32(*protocol), Audience: url, SessionTTL: *ttl, NeedsSubscription: *subscriptions, Plans: plans})
	var handler http.Handler = fake
	if *stateDir != "" {
		path := filepath.Join(*stateDir, stateFile)
		if err := load(fake, path); err != nil {
			log.Fatal(err)
		}
		handler = &keeper{next: fake, fake: fake, path: path, mu: sync.Mutex{}}
	}
	if _, err := fmt.Fprintln(os.Stdout, url); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.Serve(listener))
}
