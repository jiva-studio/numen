package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const (
	// identityFile holds the words, the chain and the keys of the computers; the acceptance harness reads it.
	identityFile = "identity.json"
	// purpose is written into identityFile and marks the keys as keys of tests.
	purpose = "TEST ONLY: keys of one acceptance scenario on a test sync server, never used for a real identity."

	keyName       = "identity"
	highWaterName = "chain-length-"

	// entrySpacing keeps every entry of the chain a minute apart and the last one a minute old, inside the clock skew the server allows.
	entrySpacing = time.Minute

	subscribePeriod  = 30 * 24 * time.Hour
	subscribeTimeout = 20 * time.Second
	subscribePoll    = 200 * time.Millisecond
	requestTimeout   = 30 * time.Second
	paymentBodyLimit = 1 << 10
)

func getComputerNames() []string { return []string{"first", "second", "third"} }

func getDeviceTitle(name string) string {
	return map[string]string{"first": "First computer", "second": "Second computer", "third": "Third computer"}[name]
}

type computerFile struct {
	DeviceFingerprint string            `json:"deviceFingerprint"`
	Credentials       map[string]string `json:"credentials"`
}

type seedFile struct {
	Purpose     string                  `json:"purpose"`
	Words       []string                `json:"words"`
	Fingerprint string                  `json:"fingerprint"`
	Chain       []byte                  `json:"chain"`
	Computers   map[string]computerFile `json:"computers"`
}

type seedOptions struct {
	label      string
	server     string
	out        string
	paymentURL string
	plan       string
}

func runSeed(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("seed", flag.ContinueOnError)
	var options seedOptions
	flags.StringVar(&options.label, "label", "", "name of the account: the same label makes the same words and keys")
	flags.StringVar(&options.server, "server", "", "address of the sync server, as the plugin has it in its settings")
	flags.StringVar(&options.out, "out", "", "folder that receives "+identityFile)
	flags.StringVar(&options.paymentURL, "payment", "", "control API of the payment stand-in; needed with -plan")
	flags.StringVar(&options.plan, "plan", "", "product the account buys, such as prod_basic; none when empty")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if options.label == "" || options.server == "" || options.out == "" {
		return errors.New("seed needs -label, -server and -out")
	}
	if options.plan != "" && options.paymentURL == "" {
		return errors.New("-plan needs -payment")
	}
	file, err := seedAccount(ctx, options)
	if err != nil {
		return err
	}
	if err := writeSeed(options.out, file); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, file.Fingerprint)
	return err
}

func seedAccount(ctx context.Context, options seedOptions) (seedFile, error) {
	made, err := makeIdentity(options.label, newClock().Now())
	if err != nil {
		return seedFile{}, err
	}
	client := &http.Client{Timeout: requestTimeout}
	token, err := signIn(ctx, client, options.server, made.devices["first"], made.first)
	if err != nil {
		return seedFile{}, fmt.Errorf("sign in the first computer: %w", err)
	}
	getToken := func() string { return token }
	if _, err := syncclient.NewAccountClient(options.server, client, getToken).AppendChain(ctx, made.chain.Bytes()); err != nil {
		return seedFile{}, fmt.Errorf("add the second and the third computer: %w", err)
	}
	if options.plan != "" {
		subscription := syncclient.NewSubscriptionClient(options.server, client, getToken)
		if err := subscribe(ctx, client, subscription, options.paymentURL, options.plan); err != nil {
			return seedFile{}, err
		}
	}
	return buildSeedFile(made)
}

func signIn(ctx context.Context, client *http.Client, server string, device identity.DeviceKey, chain identity.Chain) (string, error) {
	rpc := syncclient.NewSignInClient(server, client, newClock().Now)
	challenge, err := rpc.GetChallenge(ctx, device.Public().Sign)
	if err != nil {
		return "", err
	}
	signature := device.SignForPlugin(syncprotocol.SignInPluginID, challenge.Message)
	session, err := rpc.SignIn(ctx, syncclient.Proof{
		ChallengeID: challenge.ID, DeviceSignKey: device.Public().Sign, Signature: signature[:], Chain: chain.Bytes(), RestoreAccount: false,
	})
	if err != nil {
		return "", err
	}
	return session.Token, nil
}

func buildSeedFile(made madeIdentity) (seedFile, error) {
	words, account, chain, devices := made.words, made.account, made.chain, made.devices
	fingerprint := chain.Root().Fingerprint().String()
	file := seedFile{Purpose: purpose, Words: words.List(), Fingerprint: fingerprint, Chain: chain.Bytes(), Computers: map[string]computerFile{}}
	for _, name := range getComputerNames() {
		wrapped, err := account.WrapForDevice(devices[name].Public())
		if err != nil {
			return seedFile{}, err
		}
		record := identity.Record{Device: devices[name], Wrapped: wrapped}
		file.Computers[name] = computerFile{
			DeviceFingerprint: devices[name].Public().Fingerprint().String(),
			Credentials: map[string]string{
				keyName:                     base64.RawURLEncoding.EncodeToString(record.Bytes()),
				highWaterName + fingerprint: strconv.Itoa(chain.Len()),
			},
		}
	}
	return file, nil
}

func writeSeed(folder string, file seedFile) error {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(folder, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(folder, identityFile), append(raw, '\n'), 0o600)
}

func subscribe(ctx context.Context, client *http.Client, subscription *syncclient.SubscriptionClient, control, plan string) error {
	address, err := subscription.CreateCheckout(ctx, plan)
	if err != nil {
		return fmt.Errorf("create a checkout for %s: %w", plan, err)
	}
	parsed, err := url.Parse(address)
	if err != nil || !strings.HasPrefix(parsed.Path, "/pay/") {
		return fmt.Errorf("the checkout address %q has no /pay/<id> path", address)
	}
	periodEnd := newClock().Now().Add(subscribePeriod).Unix()
	target := fmt.Sprintf("%s/pay?checkout=%s&period_end=%d", strings.TrimSuffix(control, "/"), url.QueryEscape(strings.TrimPrefix(parsed.Path, "/pay/")), periodEnd)
	if err := sendPaymentRequest(ctx, client, target); err != nil {
		return err
	}
	deadline := newClock().Now().Add(subscribeTimeout)
	for {
		got, err := subscription.GetSubscription(ctx)
		if err == nil && got.Phase == syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE && got.PlanID == plan {
			return nil
		}
		if newClock().Now().After(deadline) {
			return fmt.Errorf("the subscription to %s is not active after %s: phase %v: %w", plan, subscribeTimeout, got.Phase, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(subscribePoll):
		}
	}
}

func sendPaymentRequest(ctx context.Context, client *http.Client, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("pay the checkout: %w", err)
	}
	defer closeQuietly(response.Body)
	body, err := io.ReadAll(io.LimitReader(response.Body, paymentBodyLimit))
	if err != nil {
		return fmt.Errorf("read the answer of the control API: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("the control API of the stand-in answered %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// madeIdentity is the person of a label before the server knows it: the keys, and the chain at the time the first computer signs in and
// at the time all three computers are in it.
type madeIdentity struct {
	words   identity.RecoveryWords
	account identity.AccountKeys
	devices map[string]identity.DeviceKey
	first   identity.Chain
	chain   identity.Chain
}

func makeIdentity(label string, now time.Time) (madeIdentity, error) {
	words, err := identity.NewRecoveryWords(newLabelStream(label + " words"))
	if err != nil {
		return madeIdentity{}, err
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		return madeIdentity{}, err
	}
	account, err := identity.NewAccountKeys(newLabelStream(label + " account"))
	if err != nil {
		return madeIdentity{}, err
	}
	made := madeIdentity{words: words, account: account, devices: map[string]identity.DeviceKey{}, first: identity.Chain{}, chain: identity.Chain{}}
	for _, name := range getComputerNames() {
		if made.devices[name], err = identity.NewDeviceKey(newLabelStream(label + " device " + name)); err != nil {
			return madeIdentity{}, err
		}
	}
	start := now.Add(-time.Duration(len(getComputerNames())+1) * entrySpacing)
	made.chain, err = identity.NewChain(root, account, start)
	if err != nil {
		return madeIdentity{}, err
	}
	for i, name := range getComputerNames() {
		made.chain, err = made.chain.AppendDevice(root, made.devices[name].Public(), getDeviceTitle(name), start.Add(time.Duration(i+1)*entrySpacing))
		if err != nil {
			return madeIdentity{}, err
		}
		if i == 0 {
			made.first = made.chain
		}
	}
	return made, nil
}
