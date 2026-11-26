package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

func runListDevices(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("list-devices", flag.ContinueOnError)
	server := flags.String("server", "", "address of the sync server, as the plugin has it in its settings")
	words := flags.String("words", "", "the recovery words of the account, separated by spaces")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *server == "" || *words == "" {
		return errors.New("list-devices needs -server and -words")
	}
	parsed, err := identity.ParseRecoveryWords(strings.Fields(*words))
	if err != nil {
		return err
	}
	root, err := parsed.DeriveRecoveryKey()
	if err != nil {
		return err
	}
	fingerprints, err := readDevices(ctx, *server, root)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(fingerprints)
}

func readDevices(ctx context.Context, server string, root identity.RecoveryKey) ([]string, error) {
	client := &http.Client{Timeout: requestTimeout}
	rootSignKey := root.Public().Sign
	challenge, err := syncclient.NewSignInClient(server, client, time.Now).GetChainReadChallenge(ctx, rootSignKey)
	if err != nil {
		return nil, err
	}
	signature := root.SignForRecovery(syncprotocol.SignInPluginID, challenge.Message)
	raw, found, err := syncclient.NewAccountClient(server, client, func() string { return "" }).GetAccountChainByRoot(ctx, challenge.ID, rootSignKey, signature[:])
	if err != nil {
		return nil, err
	}
	fingerprints := []string{}
	if !found {
		return fingerprints, nil
	}
	chain, err := identity.ParseChain(raw)
	if err != nil {
		return nil, fmt.Errorf("read the chain the server holds: %w", err)
	}
	for _, device := range chain.ListDevices() {
		fingerprints = append(fingerprints, device.Public.Fingerprint().String())
	}
	return fingerprints, nil
}
