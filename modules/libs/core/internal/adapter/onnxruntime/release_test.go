package onnxruntime

import "testing"

func TestEveryPublishedReleaseIsPinned(t *testing.T) {
	if len(releases) == 0 {
		t.Fatal("no release is published")
	}
	for platform, release := range releases {
		if err := release.check(); err != nil {
			t.Errorf("%s: %v", platform, err)
		}
	}
}

func TestNoReleaseIsPublishedForAPlatformThatHasNone(t *testing.T) {
	if _, err := getRelease("plan9", "mips"); err == nil {
		t.Fatal("getRelease() found a release for plan9/mips")
	}
}

func TestParseProvider(t *testing.T) {
	for name, want := range map[string]Provider{"": ProviderCPU, "cpu": ProviderCPU, "CUDA": ProviderCUDA, "coreml": ProviderCoreML} {
		if got, err := ParseProvider(name); err != nil || got != want {
			t.Errorf("ParseProvider(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
	if _, err := ParseProvider("directml"); err == nil {
		t.Error("ParseProvider() accepted directml")
	}
}

func TestAProviderRunsWhereItsHardwareIs(t *testing.T) {
	for _, one := range []struct {
		provider     Provider
		goos, goarch string
		ok           bool
	}{
		{ProviderCPU, "plan9", "mips", true},
		{ProviderCUDA, "linux", "amd64", true},
		{ProviderCUDA, "windows", "amd64", true},
		{ProviderCUDA, "darwin", "arm64", false},
		{ProviderCoreML, "darwin", "arm64", true},
		{ProviderCoreML, "darwin", "amd64", false},
		{ProviderCoreML, "linux", "arm64", false},
	} {
		if err := one.provider.check(one.goos, one.goarch); (err == nil) != one.ok {
			t.Errorf("%s on %s/%s = %v, want ok %v", one.provider, one.goos, one.goarch, err, one.ok)
		}
	}
}
