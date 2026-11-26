package synctest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// EnvOwnStack names the compose project of a test stack that only the tests of one package use, one test at a time. The tests that roll the database back,
// issue a restore epoch or age the deletion records act on every account of the stack, so they need a stack that no other test shares. EnvURL and the other
// variables of the stack must point at that same stack.
const EnvOwnStack = "NUMEN_SYNC_OWN_STACK"

const (
	ownDatabase    = "numen_sync"
	ownOwner       = "numen_sync_owner"
	ownCommandWait = 2 * time.Minute
	ownStartWait   = 90 * time.Second
)

var ownEpochLine = regexp.MustCompile(`restore epoch issued: epoch ([0-9a-f]{32})`)

// OwnStack drives a test stack through docker compose: it takes the database apart and puts it back, which the proxy in front of the server cannot do.
type OwnStack struct {
	Project string
	Stack   *Stack
}

// Snapshot is a dump of the database of an own stack.
type Snapshot struct {
	file string
}

// GetOwnStack returns the own stack named by EnvOwnStack. The database goes back to its state at the call when the test ends, so no test leaves an epoch, an account or a pruned record to the next. The test skips when the variable is not set, and fails then when EnvRequired is 1. A test that uses
// it must not call t.Parallel, and the packages that use it run one at a time (go test -p 1).
func GetOwnStack(t *testing.T) *OwnStack {
	t.Helper()
	project := os.Getenv(EnvOwnStack)
	if project == "" {
		if os.Getenv(EnvRequired) != "1" {
			t.Skip("no own sync stack: set " + EnvOwnStack + " to the compose project of a stack that no other test uses")
		}
		t.Fatalf("no own sync stack: set %s to the compose project of a stack that no other test uses, and %s to its address", EnvOwnStack, EnvURL)
	}
	o := &OwnStack{Project: project, Stack: GetStack(t)}
	o.checkServes(t)
	baseline := o.CreateSnapshot(t)
	t.Cleanup(func() { o.RollBack(t, baseline) })
	return o
}

// GetOwn returns the own stack behind the server of the test.
func (s *Server) GetOwn(t *testing.T) *OwnStack {
	t.Helper()
	o := GetOwnStack(t)
	if s.Stack != nil && s.Stack.URL != o.Stack.URL {
		t.Fatalf("the server of the test is at %s, the own stack at %s", s.Stack.URL, o.Stack.URL)
	}
	return o
}

// CreateSnapshot dumps the database. The dump is deleted when the test ends.
func (o *OwnStack) CreateSnapshot(t *testing.T) Snapshot {
	t.Helper()
	id := make([]byte, 8)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{file: "/tmp/numen-test-" + hex.EncodeToString(id) + ".dump"}
	o.run(t, "", "exec", "-T", "postgres", "pg_dump", "-U", "postgres", "--format=custom", "--file", snapshot.file, ownDatabase)
	t.Cleanup(func() { o.runQuietly("exec", "-T", "postgres", "rm", "-f", snapshot.file) })
	return snapshot
}

// RollBack stops the server, replaces the database with the snapshot and starts the server again, as the restore of a backup does. Every account goes
// back to the state of the snapshot.
func (o *OwnStack) RollBack(t *testing.T, snapshot Snapshot) {
	t.Helper()
	o.run(t, "", "stop", "--timeout", "2", "sync")
	o.runSQL(t, "DROP DATABASE IF EXISTS "+ownDatabase+" WITH (FORCE)", "postgres")
	o.runSQL(t, "CREATE DATABASE "+ownDatabase+" OWNER "+ownOwner, "postgres")
	o.run(t, "", "exec", "-T", "postgres", "pg_restore", "-U", "postgres", "--exit-on-error", "--dbname", ownDatabase, snapshot.file)
	o.run(t, "", "start", "sync")
	o.waitHealthy(t)
}

// IssueRestoreEpoch makes the server sign and store a restore epoch, as the operator does after a restore, and returns its id.
func (o *OwnStack) IssueRestoreEpoch(t *testing.T) []byte {
	t.Helper()
	out := o.run(t, "", "exec", "-T", "sync", "/numen-sync", "admin", "issue-restore-epoch")
	found := ownEpochLine.FindSubmatch(out)
	if found == nil {
		t.Fatalf("issue-restore-epoch printed %q, want a line with the id of the epoch", out)
	}
	id, err := hex.DecodeString(string(found[1]))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// DeleteAccountData removes the vaults, the catalogs and the usage of every account and keeps the accounts, their device chains and their subscriptions, as the
// retention job of the server does for an account whose paid period ended long ago.
func (o *OwnStack) DeleteAccountData(t *testing.T) {
	t.Helper()
	o.runSQL(t, "DELETE FROM vaults; DELETE FROM account_log; UPDATE account_usage SET used_bytes = 0", ownDatabase)
}

// ForgetRemovals ages every deletion record past its life of 90 days and has the server prune them, so that a device that was away has to read the vault
// again from the start.
func (o *OwnStack) ForgetRemovals(t *testing.T) {
	t.Helper()
	if err := o.TryForgetRemovals(); err != nil {
		t.Fatal(err)
	}
}

// TryForgetRemovals is ForgetRemovals for a caller that is not the goroutine of the test, such as a hook of the proxy.
func (o *OwnStack) TryForgetRemovals() error {
	if _, err := o.runE("UPDATE files SET committed_at = now() - interval '91 days' WHERE deleted", "exec", "-T", "postgres", "psql", "-U", "postgres", "--set", "ON_ERROR_STOP=1", "--quiet", "--dbname", ownDatabase); err != nil {
		return err
	}
	_, err := o.runE("", "exec", "-T", "sync", "/numen-sync", "admin", "prune-tombstones")
	return err
}

// EndSubscriptions moves the end of the paid period of every account to the duration ago, so that the server finds each account in the phase that long after the end. A negative duration puts the end into the future, as a new purchase does.
func (o *OwnStack) EndSubscriptions(t *testing.T, ago time.Duration) {
	t.Helper()
	o.runSQL(t, fmt.Sprintf("UPDATE entitlements SET ends_at = now() - interval '%d seconds', period_end = now() - interval '%d seconds'", int(ago.Seconds()), int(ago.Seconds())), ownDatabase)
}

func (o *OwnStack) runSQL(t *testing.T, statements, database string) {
	t.Helper()
	o.run(t, statements, "exec", "-T", "postgres", "psql", "-U", "postgres", "--set", "ON_ERROR_STOP=1", "--quiet", "--dbname", database)
}

func (o *OwnStack) run(t *testing.T, stdin string, args ...string) []byte {
	t.Helper()
	out, err := o.runE(stdin, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (o *OwnStack) runE(stdin string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ownCommandWait)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", o.Project}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(stdin), &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker compose -p %s %s: %w\n%s", o.Project, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func (o *OwnStack) runQuietly(args ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), ownCommandWait)
	defer cancel()
	_ = exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", o.Project}, args...)...).Run()
}

func (o *OwnStack) waitHealthy(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(ownStartWait)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := exec.CommandContext(ctx, "docker", "compose", "-p", o.Project, "exec", "-T", "sync", "/numen-sync", "healthcheck").Run()
		cancel()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the server of %s is not healthy %s after the start: %v", o.Project, ownStartWait, err)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// checkServes fails the test unless the payment container of the project publishes the port of the address the tests use, so that a stack that is not the
// one in the environment is never rolled back.
func (o *OwnStack) checkServes(t *testing.T) {
	t.Helper()
	address, err := url.Parse(o.Stack.URL)
	if err != nil {
		t.Fatalf("%s is not an address: %v", EnvURL, err)
	}
	port, err := strconv.Atoi(address.Port())
	if err != nil {
		t.Fatalf("%s has no port: %s", EnvURL, o.Stack.URL)
	}
	out := o.run(t, "", "ps", "payment", "--format", "json")
	for line := range bytes.SplitSeq(out, []byte("\n")) {
		var container struct {
			Publishers []struct{ PublishedPort int }
		}
		if json.Unmarshal(line, &container) != nil {
			continue
		}
		for _, publisher := range container.Publishers {
			if publisher.PublishedPort == port {
				return
			}
		}
	}
	t.Fatalf("the project %s of %s does not publish port %d of %s: the tests would roll back another stack", o.Project, EnvOwnStack, port, EnvURL)
}
