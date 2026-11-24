// Command epochkey is the offline tool of the operator of a sync server: it makes the root key whose public half is
// built into clients, signs the certificate that delegates the online epoch key of the server, and checks one.
// Run it on a machine that is not the server; the root key file never leaves it.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
)

type certFile struct {
	EpochKey      string `json:"epoch_key"`
	Serial        uint64 `json:"serial"`
	NotBefore     int64  `json:"not_before"`
	NotAfter      int64  `json:"not_after"`
	RootSignature string `json:"root_signature"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, "epochkey:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer, now func() time.Time) error {
	if len(args) == 0 {
		return errors.New("usage: epochkey keygen|sign-cert|verify-cert")
	}
	switch args[0] {
	case "keygen":
		return runKeygen(args[1:], out)
	case "sign-cert":
		return runSignCert(args[1:], out, now)
	case "verify-cert":
		return runVerifyCert(args[1:], out, now)
	}
	return fmt.Errorf("unknown command %q", args[0])
}

func runKeygen(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("keygen", flag.ContinueOnError)
	path := flags.String("out", "", "file for the private root key, created with mode 0600")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("keygen needs -out")
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(*path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(hex.EncodeToString(private.Seed()) + "\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, hex.EncodeToString(public))
	return err
}

func readRootKey(path string) (ed25519.PrivateKey, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(text)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("the root key file does not hold 32 bytes of hex")
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

func runSignCert(args []string, out io.Writer, now func() time.Time) error {
	flags := flag.NewFlagSet("sign-cert", flag.ContinueOnError)
	rootPath := flags.String("root", "", "file of the private root key")
	epochHex := flags.String("epoch-key", "", "public epoch key of the server, hex")
	serial := flags.Uint64("serial", 0, "serial of the certificate, 1 or more; a new certificate takes a higher one")
	days := flags.Int("days", 0, "validity in days from now")
	if err := flags.Parse(args); err != nil {
		return err
	}
	root, err := readRootKey(*rootPath)
	if err != nil {
		return err
	}
	epoch, err := hex.DecodeString(*epochHex)
	if err != nil || len(epoch) != ed25519.PublicKeySize {
		return errors.New("-epoch-key is not 32 bytes of hex")
	}
	if *days <= 0 {
		return errors.New("-days must be positive")
	}
	start := now()
	cert, err := epochtrust.SignCert(root, epoch, *serial, start, start.AddDate(0, 0, *days))
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(certFile{
		EpochKey: hex.EncodeToString(cert.EpochKey), Serial: cert.Serial, NotBefore: cert.NotBefore, NotAfter: cert.NotAfter,
		RootSignature: hex.EncodeToString(cert.RootSignature),
	})
}

func runVerifyCert(args []string, out io.Writer, now func() time.Time) error {
	flags := flag.NewFlagSet("verify-cert", flag.ContinueOnError)
	rootHex := flags.String("root-public", "", "public root key, hex")
	path := flags.String("cert", "", "file of the certificate as sign-cert printed it")
	if err := flags.Parse(args); err != nil {
		return err
	}
	root, err := hex.DecodeString(*rootHex)
	if err != nil || len(root) != ed25519.PublicKeySize {
		return errors.New("-root-public is not 32 bytes of hex")
	}
	text, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var file certFile
	if err := json.Unmarshal(text, &file); err != nil {
		return err
	}
	epoch, errEpoch := hex.DecodeString(file.EpochKey)
	signature, errSignature := hex.DecodeString(file.RootSignature)
	if err := errors.Join(errEpoch, errSignature); err != nil {
		return err
	}
	cert := epochtrust.Cert{EpochKey: epoch, Serial: file.Serial, NotBefore: file.NotBefore, NotAfter: file.NotAfter, RootSignature: signature}
	if err := epochtrust.VerifyCert([]ed25519.PublicKey{root}, cert, now()); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "ok: serial %d, valid until %s\n", cert.Serial, time.Unix(cert.NotAfter, 0).UTC().Format(time.RFC3339))
	return err
}
