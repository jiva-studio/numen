// Command fakeopenai serves the fake OpenAI API on a local port for the end-to-end tests. It prints
// the address it listens on, then serves until it is stopped.
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

	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func main() {
	address := flag.String("addr", "127.0.0.1:0", "address to listen on")
	key := flag.String("key", "", "bearer token every request must carry")
	dimensions := flag.Int("dimensions", fakeopenai.DefaultDimensions, "width of the embedding vectors")
	status := flag.Int("status", 0, "status that chat completions and embeddings answer with")
	retryAfter := flag.String("retry-after", "", "Retry-After header sent with the status")
	flag.Parse()

	fake := fakeopenai.New(fakeopenai.Config{APIKey: *key, Dimensions: *dimensions, Models: nil, IsLabeled: false, EmbeddingModels: nil})
	if *status != 0 {
		fake.Fail("/chat/completions", *status, *retryAfter)
		fake.Fail("/embeddings", *status, *retryAfter)
	}
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", *address)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fmt.Fprintf(os.Stdout, "http://%s/v1\n", listener.Addr()); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Handler: fake, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.Serve(listener))
}
