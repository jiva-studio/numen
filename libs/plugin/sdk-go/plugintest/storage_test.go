package plugintest

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
)

func TestGrantStorageSetsTheDirectoriesThePluginReads(t *testing.T) {
	cacheDir, dataDir := GrantStorage(t)

	gotCache, cacheErr := cache.GetPluginDir()
	gotData, dataErr := cache.GetDataDir()

	if cacheErr != nil || dataErr != nil {
		t.Fatalf("got errors %v and %v", cacheErr, dataErr)
	}
	if gotCache != cacheDir || gotData != dataDir || cacheDir == dataDir {
		t.Errorf("got %q and %q, granted %q and %q", gotCache, gotData, cacheDir, dataDir)
	}
}

func TestGrantSharedStorageSetsTheSharedDirectoryApartFromThePrivateOnes(t *testing.T) {
	shared := GrantSharedStorage(t)

	got, err := cache.GetSharedDir()
	cacheDir, cacheErr := cache.GetPluginDir()
	dataDir, dataErr := cache.GetDataDir()
	if cacheErr != nil || dataErr != nil {
		t.Fatal(cacheErr, dataErr)
	}

	if err != nil || got != shared || shared == cacheDir || shared == dataDir {
		t.Errorf("shared %q (%v), cache %q, data %q", got, err, cacheDir, dataDir)
	}
}
