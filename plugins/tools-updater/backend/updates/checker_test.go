package updates

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-updater/backend/release"
)

var errOffline = errors.New("offline")

// fakeFetcher answers every request with one manifest or one error, and records the endpoints requested.
type fakeFetcher struct {
	manifest  release.Manifest
	err       error
	requested []string
}

func (f *fakeFetcher) Fetch(_ context.Context, endpoint string) (release.Manifest, error) {
	f.requested = append(f.requested, endpoint)
	return f.manifest, f.err
}

func newManifest() release.Manifest {
	return release.Manifest{
		Version: "2026.10.1", Build: 950, Commit: "abc1234", Stamp: "", Channel: "stable",
		Notes: "", Released: "2026-10-15T12:00:00Z",
		Files: []release.File{
			{Name: "numen-windows-setup.exe", Platform: "windows", Arch: "amd64", Kind: "exe", URL: "https://dl.numen.md/setup.exe", Size: 10, SHA256: "hash"},
		},
	}
}

func windowsApp(build int64) App {
	return App{Version: "2026.9.0", Build: build, Platform: "windows"}
}

func TestACheckBeforeConfigureIsNotConfigured(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	c := NewChecker(fetcher)

	_, err := c.Check(t.Context(), windowsApp(900))

	if !errors.Is(err, ErrNotConfigured) {
		t.Errorf("error = %v, want ErrNotConfigured", err)
	}
	if len(fetcher.requested) != 0 {
		t.Errorf("the endpoint was requested before Configure: %v", fetcher.requested)
	}
}

func TestACheckOfANewerBuildReportsTheUpdate(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	c := NewChecker(fetcher)
	c.Configure(map[string]string{})

	status, err := c.Check(t.Context(), windowsApp(900))

	if err != nil {
		t.Fatal(err)
	}
	if status.State != StateAvailable || status.Update == nil {
		t.Fatalf("status = %+v", status)
	}
	if status.Update.Version != "2026.10.1" || status.Update.DownloadURL != "https://dl.numen.md/setup.exe" {
		t.Errorf("update = %+v", status.Update)
	}
	if status.Update.File == nil || status.Update.File.SHA256 != "hash" {
		t.Errorf("file = %+v", status.Update.File)
	}
	if len(fetcher.requested) != 1 || fetcher.requested[0] != StableEndpoint {
		t.Errorf("endpoints requested = %v, want the stable feed", fetcher.requested)
	}

	devStatus, err := c.Check(t.Context(), windowsApp(0))
	if err != nil || devStatus.State != StateAvailable {
		t.Errorf("dev build status = %+v, error = %v", devStatus, err)
	}
}

func TestACheckOfTheCurrentBuildReportsNoUpdate(t *testing.T) {
	c := NewChecker(&fakeFetcher{manifest: newManifest()})
	c.Configure(map[string]string{})

	status, err := c.Check(t.Context(), windowsApp(950))

	if err != nil || status.State != StateCurrent || status.Update != nil {
		t.Errorf("status = %+v, error = %v", status, err)
	}
}

func TestAnOfflineCheckIsNoFailure(t *testing.T) {
	c := NewChecker(&fakeFetcher{err: errOffline})
	c.Configure(map[string]string{})

	status, err := c.Check(t.Context(), windowsApp(900))

	if err != nil || status.State != StateUnreachable {
		t.Errorf("status = %+v, error = %v", status, err)
	}
}

func TestACheckOfAnUnknownBuildIsRefused(t *testing.T) {
	c := NewChecker(&fakeFetcher{manifest: newManifest()})
	c.Configure(map[string]string{})
	cases := map[string]struct {
		app  App
		want func(error) bool
	}{
		"no platform":      {App{Version: "2026.9.0", Build: 1, Platform: ""}, isPlatformError},
		"another platform": {App{Version: "2026.9.0", Build: 1, Platform: "plan9"}, isPlatformError},
		"no version":       {App{Version: " ", Build: 1, Platform: "linux"}, isUnknownVersion},
		"a negative build": {App{Version: "2026.9.0", Build: -1, Platform: "linux"}, isUnknownVersion},
	}
	for name, c2 := range cases {
		_, err := c.Check(t.Context(), c2.app)

		if !c2.want(err) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func isPlatformError(err error) bool {
	failure, ok := errors.AsType[*PlatformError](err)
	return ok && failure != nil
}

func isUnknownVersion(err error) bool { return errors.Is(err, ErrUnknownVersion) }

func TestTheStatusIsTheResultOfTheLastCheck(t *testing.T) {
	c := NewChecker(&fakeFetcher{manifest: newManifest()})
	c.Configure(map[string]string{})
	if before := c.GetStatus(); before.State != StateUnchecked {
		t.Fatalf("status before a check = %+v", before)
	}

	if _, err := c.Check(t.Context(), windowsApp(900)); err != nil {
		t.Fatal(err)
	}

	if after := c.GetStatus(); after.State != StateAvailable {
		t.Errorf("status after a check = %+v", after)
	}
}

func TestAChangeOfTheEndpointDiscardsTheLastResult(t *testing.T) {
	c := NewChecker(&fakeFetcher{manifest: newManifest()})
	c.Configure(map[string]string{"channel": "stable"})
	if _, err := c.Check(t.Context(), windowsApp(900)); err != nil {
		t.Fatal(err)
	}

	c.Configure(map[string]string{"channel": "stable", "checkIntervalHours": "1"})
	if same := c.GetStatus(); same.State != StateAvailable {
		t.Fatalf("status after a setting that is not the endpoint = %+v", same)
	}

	c.Configure(map[string]string{"channel": "beta"})
	if changed := c.GetStatus(); changed.State != StateUnchecked {
		t.Errorf("status after a new endpoint = %+v", changed)
	}
}

func TestTheCustomChannelWithoutAnAddressAsksNoFeedAndIsUnreachable(t *testing.T) {
	fetcher := &fakeFetcher{manifest: newManifest()}
	c := NewChecker(fetcher)
	c.Configure(map[string]string{"channel": "custom", "endpoint": " "})

	status, err := c.Check(t.Context(), windowsApp(900))

	if err != nil || status.State != StateUnreachable {
		t.Errorf("status = %+v, error = %v", status, err)
	}
	if len(fetcher.requested) != 0 {
		t.Errorf("endpoints requested = %v, want none", fetcher.requested)
	}
}
