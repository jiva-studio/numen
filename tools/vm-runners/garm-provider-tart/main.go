package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	exitCodeNotFound  = 30
	exitCodeDuplicate = 31
	interfaceVersion  = "v0.1.0"
)

func main() {
	os.Exit(runProvider(context.Background(), os.Getenv, os.Stdin, os.Stdout, os.Stderr, execRunner{}))
}

func runProvider(ctx context.Context, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, runner commandRunner) int {
	err := executeCommand(ctx, getenv, stdin, stdout, runner)
	if err == nil {
		return 0
	}
	if _, writeErr := fmt.Fprintf(stderr, "garm-provider-tart: %s\n", err); writeErr != nil {
		return 1
	}
	switch {
	case errors.Is(err, errNotFound):
		return exitCodeNotFound
	case errors.Is(err, errDuplicate):
		return exitCodeDuplicate
	default:
		return 1
	}
}

func executeCommand(ctx context.Context, getenv func(string) string, stdin io.Reader, stdout io.Writer, runner commandRunner) error {
	command := getenv("GARM_COMMAND")
	if command == "GetVersion" {
		_, err := fmt.Fprintln(stdout, interfaceVersion)
		return err
	}
	p, err := createProvider(getenv, runner)
	if err != nil {
		return err
	}
	instanceID := getenv("GARM_INSTANCE_ID")
	switch command {
	case "CreateInstance":
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("read bootstrap params: %w", err)
		}
		return writeResult[providerInstance](stdout)(p.createInstance(ctx, raw))
	case "DeleteInstance":
		return p.deleteInstance(ctx, instanceID)
	case "GetInstance":
		return writeResult[providerInstance](stdout)(p.getInstance(ctx, instanceID))
	case "ListInstances":
		return writeResult[[]providerInstance](stdout)(p.listInstances(ctx, getenv("GARM_POOL_ID")))
	case "RemoveAllInstances":
		return p.removeAllInstances(ctx)
	case "StartInstance":
		return p.startInstance(ctx, instanceID)
	case "StopInstance":
		return p.stopInstance(ctx, instanceID)
	default:
		return fmt.Errorf("unknown GARM_COMMAND %q", command)
	}
}

func createProvider(getenv func(string) string, runner commandRunner) (provider, error) {
	cfg, err := readConfig(getenv("GARM_PROVIDER_CONFIG_FILE"))
	if err != nil {
		return provider{}, err
	}
	controllerID := getenv("GARM_CONTROLLER_ID")
	if controllerID == "" {
		return provider{}, errors.New("GARM_CONTROLLER_ID is not set")
	}
	return provider{
		cfg:             cfg,
		tart:            tartClient{runner: runner, binary: cfg.Tart},
		processes:       processTable{runner: runner},
		store:           stateStore{dir: cfg.StateDir},
		controllerID:    controllerID,
		exitWaitSeconds: 10,
	}, nil
}

func writeResult[T any](w io.Writer) func(T, error) error {
	return func(value T, err error) error {
		if err != nil {
			return err
		}
		return writeJSON(w, value)
	}
}

func writeJSON(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode output: %w", err)
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
