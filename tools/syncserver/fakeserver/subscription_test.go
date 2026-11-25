package fakeserver_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

const day = 24 * time.Hour

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type billing struct {
	tb           testing.TB
	server       *fakeserver.Server
	clock        *testClock
	address      string
	env          *env
	subscription *syncclient.SubscriptionClient
}

func newBillingServer(tb testing.TB, clock *testClock, needsSubscription bool) (*fakeserver.Server, string) {
	tb.Helper()
	server := fakeserver.New(fakeserver.Config{Now: clock.Now, SessionTTL: 1000 * day, NeedsSubscription: needsSubscription})
	served := httptest.NewServer(server)
	tb.Cleanup(served.Close)
	server.SetAudience(served.URL)
	return server, served.URL
}

func createBillingEnv(tb testing.TB, needsSubscription bool) *billing {
	tb.Helper()
	clock := &testClock{now: machineClock{}.Now()}
	server, address := newBillingServer(tb, clock, needsSubscription)
	account := newOwnedAccount(tb)
	device := account.addDevice(tb, "First")
	return newBillingOf(tb, server, clock, address, &account, device)
}

func newBillingOf(tb testing.TB, server *fakeserver.Server, clock *testClock, address string, account *ownedAccount, device identity.DeviceKey) *billing {
	tb.Helper()
	e := newEnvOn(tb, address, account, device)
	return &billing{
		tb: tb, server: server, clock: clock, address: address, env: e,
		subscription: syncclient.NewSubscriptionClient(address, http.DefaultClient, func() string { return e.token }),
	}
}

func (b *billing) phase() syncv1.SubscriptionPhase {
	b.tb.Helper()
	got, err := b.subscription.GetSubscription(b.tb.Context())
	if err != nil {
		b.tb.Fatal(err)
	}
	return got.Phase
}

func (b *billing) pay() {
	b.tb.Helper()
	url, err := b.subscription.CreateCheckout(b.tb.Context(), "plus")
	if err != nil {
		b.tb.Fatal(err)
	}
	status, _ := b.get(url)
	if status != http.StatusOK {
		b.tb.Fatalf("the pay page answered %d, want 200", status)
	}
}

func (b *billing) get(url string) (int, string) {
	b.tb.Helper()
	request, err := http.NewRequestWithContext(b.tb.Context(), http.MethodGet, url, nil)
	if err != nil {
		b.tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		b.tb.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			b.tb.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		b.tb.Fatal(err)
	}
	return response.StatusCode, string(body)
}

func (b *billing) setPhase(query string) int {
	b.tb.Helper()
	request, err := http.NewRequestWithContext(b.tb.Context(), http.MethodPost, b.address+"/admin/subscription?"+query, nil)
	if err != nil {
		b.tb.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		b.tb.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			b.tb.Error(err)
		}
	}()
	return response.StatusCode
}

func (b *billing) fingerprint() string {
	return b.env.account.chain.Root().Fingerprint().String()
}

func (b *billing) canAppendCatalogEntry() error {
	b.tb.Helper()
	_, err := b.env.catalog.AppendCatalogEntry(b.tb.Context(), b.env.newCatalogEntry(newID(16), true, "vault"))
	return err
}

func TestGetSubscription_IsActiveForEveryAccountWhenTheServerDoesNotNeedASubscription(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, false)

	got, err := b.subscription.GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		t.Errorf("phase = %v, want active", got.Phase)
	}
	if want := b.clock.Now().Add(30 * day).Truncate(time.Second); !got.PeriodEnd.Equal(want) {
		t.Errorf("PeriodEnd = %v, want %v", got.PeriodEnd, want)
	}
	if err := b.canAppendCatalogEntry(); err != nil {
		t.Errorf("a write was refused: %v", err)
	}
}

func TestGetSubscription_IsNoneForANewAccountWhenTheServerNeedsASubscription(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	got, err := b.subscription.GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE || !got.PeriodEnd.IsZero() || !got.WritesEnd.IsZero() || !got.DeleteAt.IsZero() {
		t.Errorf("subscription = %+v, want phase none and zero times", got)
	}
}

func TestGetSubscription_WithoutASessionIsUnauthenticated(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	anonymous := syncclient.NewSubscriptionClient(b.address, http.DefaultClient, func() string { return "" })

	_, err := anonymous.GetSubscription(t.Context())

	if !isUnauthenticated(err) {
		t.Errorf("error = %v, want unauthenticated", err)
	}
}

func TestCreateCheckout_ReturnsAPayPageOfTheServer(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	url, err := b.subscription.CreateCheckout(t.Context(), "plus")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(url, b.address+"/pay/") || len(url) == len(b.address+"/pay/") {
		t.Errorf("url = %q, want %s/pay/ and an id", url, b.address)
	}
}

func TestPayPage_ActivatesTheSubscriptionForThirtyDays(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	url, err := b.subscription.CreateCheckout(t.Context(), "plus")
	if err != nil {
		t.Fatal(err)
	}

	status, body := b.get(url)

	if status != http.StatusOK || !strings.Contains(body, "Payment received") {
		t.Errorf("pay page = %d %q, want 200 and Payment received", status, body)
	}
	got, err := b.subscription.GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	periodEnd := b.clock.Now().Add(30 * day).Truncate(time.Second)
	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE || !got.PeriodEnd.Equal(periodEnd) ||
		!got.WritesEnd.Equal(periodEnd.Add(7*day)) || !got.DeleteAt.Equal(periodEnd.Add(60*day)) {
		t.Errorf("subscription = %+v, want active until %v with writes end 7 days and delete 60 days after", got, periodEnd)
	}
}

func TestPayPage_AnswersNotFoundForAnUnknownId(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	status, _ := b.get(b.address + "/pay/unknown")

	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		t.Errorf("phase = %v, want none", got)
	}
}

func TestPayPage_OfAnotherAccountDoesNotActivateThisAccount(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	other := newSecondAccountOn(t, b.address)
	otherClient := syncclient.NewSubscriptionClient(b.address, http.DefaultClient, func() string { return other.token })
	url, err := otherClient.CreateCheckout(t.Context(), "plus")
	if err != nil {
		t.Fatal(err)
	}

	b.get(url)

	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		t.Errorf("phase = %v, want none", got)
	}
}

func TestCreateCheckout_IsAlreadySubscribedWhileTheSubscriptionIsActive(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.pay()

	_, err := b.subscription.CreateCheckout(t.Context(), "plus")

	if !isAlreadySubscribed(err) {
		t.Errorf("error = %v, want ErrAlreadySubscribed", err)
	}
}

func TestCreateCheckout_IsOfferedAgainInEveryPhaseThatIsNotActive(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"none", "grace", "read_only", "deleted"} {
		b := createBillingEnv(t, true)
		b.pay()
		if status := b.setPhase("phase=" + phase); status != http.StatusNoContent && status != http.StatusOK {
			t.Fatalf("admin phase %s answered %d", phase, status)
		}

		if _, err := b.subscription.CreateCheckout(t.Context(), "plus"); err != nil {
			t.Errorf("CreateCheckout in phase %s: %v, want a url", phase, err)
		}
	}
}

func TestSubscription_BelongsToTheAccountAndNotToTheSession(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	second := b.env.account.addDevice(t, "Second")
	other := newBillingOf(t, b.server, b.clock, b.address, b.env.account, second)

	b.pay()

	if got := other.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		t.Errorf("phase on the second device = %v, want active", got)
	}
}

func TestSubscription_OfOneAccountDoesNotReachAnotherAccount(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	account := newOwnedAccount(t)
	device := account.addDevice(t, "Other")
	other := newBillingOf(t, b.server, b.clock, b.address, &account, device)

	b.pay()

	if got := other.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		t.Errorf("phase of the other account = %v, want none", got)
	}
}

func TestWrites_AreRefusedWithoutASubscriptionAndAllowedAfterPayment(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	if err := b.canAppendCatalogEntry(); !isWriteNotAllowed(err) {
		t.Fatalf("a write without a subscription: error = %v, want ErrWriteNotAllowed", err)
	}
	b.pay()

	if err := b.canAppendCatalogEntry(); err != nil {
		t.Errorf("a write after payment: %v, want it accepted", err)
	}
}

func TestPhaseArithmetic_FollowsTheBoundariesOfTheGraceAndTheRetention(t *testing.T) {
	t.Parallel()
	for name, one := range map[string]struct {
		after    time.Duration
		phase    syncv1.SubscriptionPhase
		canWrite bool
	}{
		"the last second of the period":    {after: 30*day - time.Second, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, canWrite: true},
		"the end of the period":            {after: 30 * day, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, canWrite: true},
		"one hour before the end of grace": {after: 36*day + 23*time.Hour, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, canWrite: true},
		"seven days after the period":      {after: 37 * day, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, canWrite: false},
		"the last second of the retention": {after: 90*day - time.Second, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, canWrite: false},
		"the end of the retention":         {after: 90 * day, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, canWrite: false},
		"one second after the retention":   {after: 90*day + time.Second, phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, canWrite: false},
	} {
		b := createBillingEnv(t, true)
		b.pay()
		b.clock.advance(one.after)

		if got := b.phase(); got != one.phase {
			t.Errorf("%s: phase = %v, want %v", name, got, one.phase)
		}
		err := b.canAppendCatalogEntry()
		if one.canWrite && err != nil {
			t.Errorf("%s: a write failed: %v", name, err)
		}
		if !one.canWrite && !isWriteNotAllowed(err) {
			t.Errorf("%s: a write: error = %v, want ErrWriteNotAllowed", name, err)
		}
	}
}

func TestAdminSubscription_SetsThePhaseWithTheAgeOfTheEnd(t *testing.T) {
	t.Parallel()
	for name, one := range map[string]struct {
		query     string
		phase     syncv1.SubscriptionPhase
		periodEnd time.Duration
	}{
		"active":    {query: "phase=active", phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE, periodEnd: 30 * day},
		"grace":     {query: "phase=grace", phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE, periodEnd: -1 * day},
		"read only": {query: "phase=read_only", phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY, periodEnd: -10 * day},
		"deleted":   {query: "phase=deleted", phase: syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED, periodEnd: -70 * day},
	} {
		b := createBillingEnv(t, true)

		if status := b.setPhase(one.query); status >= 300 {
			t.Fatalf("%s: admin answered %d", name, status)
		}

		got, err := b.subscription.GetSubscription(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got.Phase != one.phase || !got.PeriodEnd.Equal(b.clock.Now().Add(one.periodEnd).Truncate(time.Second)) {
			t.Errorf("%s: subscription = %+v, want phase %v ending at now %+v", name, got, one.phase, one.periodEnd)
		}
	}
}

func TestAdminSubscription_NoneRemovesTheSubscription(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.pay()

	if status := b.setPhase("phase=none"); status >= 300 {
		t.Fatalf("admin answered %d", status)
	}

	got, err := b.subscription.GetSubscription(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE || !got.PeriodEnd.IsZero() {
		t.Errorf("subscription = %+v, want none", got)
	}
}

func TestAdminSubscription_WithoutAnAccountSetsEveryAccount(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	account := newOwnedAccount(t)
	device := account.addDevice(t, "Other")
	other := newBillingOf(t, b.server, b.clock, b.address, &account, device)

	b.setPhase("phase=read_only")

	if b.phase() != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY || other.phase() != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY {
		t.Error("an account stayed out of the phase, want both read only")
	}
}

func TestAdminSubscription_WithAnAccountSetsOnlyThatAccount(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	account := newOwnedAccount(t)
	device := account.addDevice(t, "Other")
	other := newBillingOf(t, b.server, b.clock, b.address, &account, device)

	b.setPhase("phase=read_only&account=" + strings.ReplaceAll(b.fingerprint(), " ", "%20"))

	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_READ_ONLY {
		t.Errorf("phase of the named account = %v, want read only", got)
	}
	if got := other.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		t.Errorf("phase of the other account = %v, want none", got)
	}
}

func TestAdminSubscription_RefusesAnUnknownPhaseAndAWrongMethod(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)

	if status := b.setPhase("phase=sleeping"); status != http.StatusBadRequest {
		t.Errorf("unknown phase answered %d, want 400", status)
	}
	if status, _ := b.get(b.address + "/admin/subscription?phase=active"); status != http.StatusMethodNotAllowed {
		t.Errorf("GET answered %d, want 405", status)
	}
	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_NONE {
		t.Errorf("phase = %v, want none", got)
	}
}

func TestWrites_AreRefusedInReadOnlyOnTheCatalogTheRevisionsAndTheBlobs(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.pay()
	vault := b.env.newVault()
	file := newID(16)
	revision := b.env.newFile(vault, file, nil, "Shopping list.md", []byte("milk"))
	b.setPhase("phase=read_only")

	_, appendErr := b.env.catalog.AppendCatalogEntry(t.Context(), b.env.newCatalogEntry(vault, false, "renamed"))
	_, commitErr := b.env.revision.CommitRevisions(t.Context(), vault, []*syncv1.Revision{revision})
	_, reserveErr := b.env.blob.ReserveBlobs(t.Context(), vault, []uint64{4})

	for name, err := range map[string]error{"AppendCatalogEntry": appendErr, "CommitRevisions": commitErr, "ReserveBlobs": reserveErr} {
		if !isWriteNotAllowed(err) {
			t.Errorf("%s: error = %v, want ErrWriteNotAllowed", name, err)
		}
	}
}

func TestReads_AreAllowedInEveryPhase(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"grace", "read_only", "none"} {
		b := createBillingEnv(t, true)
		b.pay()
		vault := b.env.newVault()
		body := []byte("milk")
		revision := b.env.newFile(vault, newID(16), nil, "Shopping list.md", body)
		b.env.commit(vault, revision)
		b.setPhase("phase=" + phase)

		_, catalogErr := b.env.catalog.GetCatalog(t.Context(), 0)
		changes, changesErr := b.env.revision.GetChanges(t.Context(), vault, 0, 0)
		_, urlsErr := b.env.blob.GetBlobUrls(t.Context(), vault, [][]byte{revision.GetBlobId()})

		if catalogErr != nil || changesErr != nil || urlsErr != nil {
			t.Errorf("phase %s: reads failed: %v, %v, %v", phase, catalogErr, changesErr, urlsErr)
		}
		if len(changes.GetChanges()) != 1 {
			t.Errorf("phase %s: GetChanges returned %d changes, want 1", phase, len(changes.GetChanges()))
		}
	}
}

func TestDeleted_RemovesTheVaultsOfTheAccount(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.pay()
	vault := b.env.newVault()
	b.env.commit(vault, b.env.newFile(vault, newID(16), nil, "Shopping list.md", []byte("milk")))
	b.setPhase("phase=deleted")

	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_DELETED {
		t.Fatalf("phase = %v, want deleted", got)
	}
	changes, err := b.env.revision.GetChanges(t.Context(), vault, 0, 0)
	if err == nil && len(changes.GetChanges()) != 0 {
		t.Errorf("GetChanges returned %d changes after the deletion, want none", len(changes.GetChanges()))
	}
}

func TestPayPage_InReadOnlyOpensTheWritesAgain(t *testing.T) {
	t.Parallel()
	b := createBillingEnv(t, true)
	b.pay()
	vault := b.env.newVault()
	b.setPhase("phase=read_only")
	if _, err := b.env.catalog.AppendCatalogEntry(t.Context(), b.env.newCatalogEntry(vault, false, "renamed")); !isWriteNotAllowed(err) {
		t.Fatalf("a write in read only: error = %v, want ErrWriteNotAllowed", err)
	}

	b.pay()

	if got := b.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_ACTIVE {
		t.Errorf("phase = %v, want active", got)
	}
	if _, err := b.env.catalog.AppendCatalogEntry(t.Context(), b.env.newCatalogEntry(vault, false, "renamed")); err != nil {
		t.Errorf("a write after payment: %v, want it accepted", err)
	}
}

func TestSnapshot_CarriesTheSubscriptionsOfTheAccounts(t *testing.T) {
	t.Parallel()
	clock := &testClock{now: machineClock{}.Now()}
	first, address := newBillingServer(t, clock, true)
	account := newOwnedAccount(t)
	device := account.addDevice(t, "First")
	b := newBillingOf(t, first, clock, address, &account, device)
	b.pay()
	b.setPhase("phase=grace")
	kept, err := first.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	second, secondAddress := newBillingServer(t, clock, true)

	if err := second.Restore(kept); err != nil {
		t.Fatal(err)
	}

	restored := newBillingOf(t, second, clock, secondAddress, &account, device)
	if got := restored.phase(); got != syncv1.SubscriptionPhase_SUBSCRIPTION_PHASE_GRACE {
		t.Errorf("phase after restore = %v, want grace", got)
	}
}

func isAlreadySubscribed(err error) bool { return errors.Is(err, syncprotocol.ErrAlreadySubscribed) }

func isWriteNotAllowed(err error) bool { return errors.Is(err, syncprotocol.ErrWriteNotAllowed) }
