// Command transcription serves the transcription plugin as a process.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var modelsDirFlag string
	var runtimeFlag string

	flag.StringVar(&modelsDirFlag, "models-dir", "", "Directory holding model weights")
	flag.StringVar(&runtimeFlag, "runtime", "", "Path to ONNX runtime library")

	s := server.New()
	if err := service.MountFromEnv(s, service.NewConfig(modelsDirFlag, runtimeFlag)); err != nil {
		return err
	}
	return s.Serve()
}
