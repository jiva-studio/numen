// Command fakesync serves the fake sync server on a local port for the end-to-end tests. It prints the
// address it listens on, then serves until it is stopped.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/fakesync"
)

func main() {
	address := flag.String("addr", "127.0.0.1:0", "address to listen on")
	protocol := flag.Uint("protocol", 0, "protocol version the server needs; 0 is the version of this build")
	ttl := flag.Duration("session-ttl", time.Hour, "how long a session lives")
	flag.Parse()

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + listener.Addr().String()
	fake := fakesync.New(fakesync.Config{Protocol: uint32(*protocol), Audience: url, SessionTTL: *ttl})
	if _, err := fmt.Fprintln(os.Stdout, url); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: fake, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.Serve(listener))
}
