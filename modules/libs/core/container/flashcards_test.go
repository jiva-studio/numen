package container_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A machine naming no cache folder is what the schedules say they fall back
// for, so the store they answer with has to be one a caller can test against
// nil. A typed nil pointer inside the interface passes that test and panics on
// the first call.
func TestAMachineWithNoCacheFolderIsAnsweredNoStoreAtAll(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("LocalAppData", "")

	for _, one := range []struct {
		what  string
		asked func(container.Config) (port.ScheduleStore, error)
	}{
		{"the schedules", container.Config.Schedules},
		{"the counting", container.Config.OpenDayCounts},
	} {
		kept, err := one.asked(container.Config{ServiceDir: ".numen"})
		if err == nil {
			t.Errorf("%s found a folder on a machine that names none", one.what)
		}
		if kept != nil {
			t.Errorf("%s answered %#v, which is not nothing", one.what, kept)
		}
	}
}

// How many cores this machine has is read here and nowhere else, and the
// simulator behind a preset's control is told it.
func TestTheCurvesAssembledAreToldTheMachinesCores(t *testing.T) {
	held := container.Config{ServiceDir: ".numen"}.Flashcards(nil, nil, nil, nil)
	if got := held.Curves.Cores; got != runtime.GOMAXPROCS(0) {
		t.Errorf("the curves run %d places at once, and the machine runs %d", got, runtime.GOMAXPROCS(0))
	}
}

// isolate points everything a machine keeps of its own at folders of the test's.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// hasFileUnder is whether any file stands anywhere below dir.
func hasFileUnder(t *testing.T, dir string) bool {
	t.Helper()
	found := false
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			found = true
		}
		return err
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return found
}

// A configuration naming a folder keeps the schedules and the day counts in it,
// whether or not the machine has a cache location of its own.
func TestTheSchedulesAreKeptInTheFolderTheConfigurationNames(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LocalAppData", "")
	folder := t.TempDir()
	cfg := container.Config{ServiceDir: ".numen", SchedulesPath: folder}

	schedules, err := cfg.Schedules()
	if err != nil {
		t.Fatal(err)
	}
	if err := schedules.Write(t.Context(), "vault1", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if !hasFileUnder(t, folder) {
		t.Fatalf("nothing was kept in %s", folder)
	}
	if hasFileUnder(t, filepath.Join(folder, "days")) {
		t.Error("a schedule was kept where the days are")
	}

	days, err := cfg.OpenDayCounts()
	if err != nil {
		t.Fatal(err)
	}
	if err := days.Write(t.Context(), "vault1", []byte("two")); err != nil {
		t.Fatal(err)
	}
	if !hasFileUnder(t, filepath.Join(folder, "days")) {
		t.Errorf("no day count was kept in %s", filepath.Join(folder, "days"))
	}
}

// Without a folder named, the schedules are kept in the machine's cache location.
func TestTheSchedulesAreKeptInTheMachinesCacheWhenNoFolderIsNamed(t *testing.T) {
	isolate(t)
	caches := []string{os.Getenv("XDG_CACHE_HOME"), os.Getenv("HOME")}

	schedules, err := container.Config{ServiceDir: ".numen"}.Schedules()
	if err != nil {
		t.Fatal(err)
	}
	if err := schedules.Write(t.Context(), "vault1", []byte("one")); err != nil {
		t.Fatal(err)
	}
	if !hasFileUnder(t, caches[0]) && !hasFileUnder(t, caches[1]) {
		t.Errorf("nothing was kept in %v", caches)
	}
}

// How many cores the scenarios are told is the compute plan's workers, and the
// machine's processors when the plan names none.
func TestTheScenariosAreToldTheCoresThePlanNames(t *testing.T) {
	isolate(t)
	before := runtime.GOMAXPROCS(3)
	t.Cleanup(func() { runtime.GOMAXPROCS(before) })

	for _, one := range []struct {
		what    string
		workers int
		want    int
	}{
		{"no workers", 0, 3},
		{"one worker", 1, 1},
		{"five workers", 5, 5},
	} {
		cfg := container.Config{ServiceDir: ".numen", SchedulesPath: t.TempDir()}
		cfg.ResolvedCompute.Workers = one.workers
		held := cfg.Flashcards(nil, nil, nil, nil)
		if held.CardFaces.Cores != one.want {
			t.Errorf("%s: the faces parse %d at once, want %d", one.what, held.CardFaces.Cores, one.want)
		}
		if held.Curves.Cores != one.want {
			t.Errorf("%s: the curves run %d at once, want %d", one.what, held.Curves.Cores, one.want)
		}
	}
}

type problemsStub struct{ port.ProblemQueries }

// The scenarios remember their working out in the folder the configuration
// names, and report nothing while that works.
func TestTheScenariosKeepWhatTheyWorkOutInTheNamedFolder(t *testing.T) {
	isolate(t)
	folder := t.TempDir()
	var told []error
	cfg := container.Config{
		ServiceDir:    ".numen",
		SchedulesPath: folder,
		ErrorHandler:  func(err error) { told = append(told, err) },
	}
	problems := &problemsStub{}
	held := cfg.Flashcards(nil, nil, problems, nil)

	if len(told) != 0 {
		t.Errorf("told %v, where nothing went wrong", told)
	}
	if held.Presets.Problems != port.ProblemQueries(problems) {
		t.Error("the presets are not told where a parse's problems are kept")
	}
	if held.Schedules.Cache == nil {
		t.Fatal("the schedules have nowhere to be remembered")
	}
	if err := held.Schedules.Cache.Write(t.Context(), "vault1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if !hasFileUnder(t, folder) || hasFileUnder(t, filepath.Join(folder, "days")) {
		t.Error("the schedules are not remembered in the folder named")
	}
	if held.Counted.Cache == nil {
		t.Fatal("the day counts have nowhere to be remembered")
	}
	if err := held.Counted.Cache.Write(t.Context(), "vault1", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if !hasFileUnder(t, filepath.Join(folder, "days")) {
		t.Error("the day counts are not remembered beside the schedules")
	}
}

// A machine that cannot say where its caches go reports each store it lacks,
// and the scenarios work without them.
func TestAMachineWithNoCacheFolderReportsEachStoreItLacks(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LocalAppData", "")
	var told []error
	cfg := container.Config{
		ServiceDir:   ".numen",
		ErrorHandler: func(err error) { told = append(told, err) },
	}
	held := cfg.Flashcards(nil, nil, nil, nil)

	if len(told) != 2 {
		t.Errorf("told %d things, want one for the schedules and one for the days: %v", len(told), told)
	}
	if held.Schedules.Cache != nil || held.Counted.Cache != nil {
		t.Error("a store was handed over where the machine has none")
	}
}

// The day of review begins where the settings file says it does.
func TestTheDayOfReviewBeginsWhereTheSettingsSayIt(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(file, []byte(`{"review": {"day_starts": "06:15"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	got := container.Config{SettingsPath: file}.DayStarts()
	if want := 6*time.Hour + 15*time.Minute; got != want {
		t.Errorf("the day begins %v past midnight, want %v", got, want)
	}
}

// Each store a machine lacks is reported with the reason it lacks it.
func TestEachStoreAMachineLacksIsReportedWithItsReason(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("LocalAppData", "")
	var told []error
	cfg := container.Config{
		ServiceDir:   ".numen",
		ErrorHandler: func(err error) { told = append(told, err) },
	}
	cfg.Flashcards(nil, nil, nil, nil)

	for _, err := range told {
		if errors.Unwrap(err) == nil {
			t.Errorf("%q carries no reason", err)
		}
	}
}
