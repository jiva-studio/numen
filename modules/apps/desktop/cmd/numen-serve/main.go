// numen-serve answers the interface's questions over HTTP, without a window.
//
// It exists so that the view can be worked on in a browser, against a real
// vault, without building a webview. The window runs the same handler.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

func main() {
	var cfg container.Config
	address := flag.String("address", "127.0.0.1:34115", "where to listen")
	flag.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	flag.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	api, closeIndex, err := webui.Open(ctx, cfg, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-serve:", err)
		os.Exit(1)
	}
	defer closeIndex()

	fmt.Printf("http://%s — showing %s\n", *address, api.Showing().Name)
	server := &http.Server{Addr: *address, Handler: api.Serving(webui.Pages())}
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "numen-serve:", err)
		os.Exit(1)
	}
}
