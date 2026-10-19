package container

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// modulesRoot is the directory the repository's modules sit in, and docsRoot is
// the directory of the repository's documentation, both from this package.
const (
	modulesRoot = "../../.."
	docsRoot    = "../../../../docs"
	modulesPath = "github.com/jiva-studio/numen/modules/"
)

// pluginManifest is a plugin's manifest as the tree holds it.
type pluginManifest struct {
	// Dir is the directory of the plugin.
	Dir string
	Raw []byte
	port.PluginManifest
}

// getPluginManifests reads every manifest the plugins directory holds.
func getPluginManifests(t *testing.T) []pluginManifest {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(modulesRoot, "plugins", "*", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no manifest found under the plugins: a rule checked against nothing")
	}
	var manifests []pluginManifest
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := port.ReadManifest(strings.NewReader(string(raw)))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		manifests = append(manifests, pluginManifest{Dir: filepath.Dir(path), Raw: raw, PluginManifest: parsed})
	}
	return manifests
}

// listLifecycleGaps lists what a manifest with a backend leaves out of its
// lifecycle: the idle policy and the memory limit are each declared by name.
func listLifecycleGaps(raw []byte) []string {
	var manifest struct {
		Backend   *json.RawMessage `json:"backend"`
		Lifecycle map[string]any   `json:"lifecycle"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return []string{err.Error()}
	}
	if manifest.Backend == nil {
		return nil
	}
	var gaps []string
	if _, declared := manifest.Lifecycle["idleTimeoutSeconds"].(float64); !declared {
		gaps = append(gaps, "idleTimeoutSeconds is not declared")
	}
	bytes, _ := manifest.Lifecycle["memoryLimitBytes"].(float64)
	megabytes, _ := manifest.Lifecycle["memoryLimitMB"].(float64)
	if bytes <= 0 && megabytes <= 0 {
		gaps = append(gaps, "memoryLimitBytes is not declared")
	}
	return gaps
}

func TestEveryPluginWithABackendDeclaresAMemoryLimitAndAnIdlePolicy(t *testing.T) {
	for _, one := range getPluginManifests(t) {
		for _, gap := range listLifecycleGaps(one.Raw) {
			t.Errorf("%s: %s", one.Dir, gap)
		}
	}
}

func TestWhatTheLifecycleRuleRefuses(t *testing.T) {
	for name, one := range map[string]struct {
		manifest string
		gaps     int
	}{
		"both declared":          {`{"backend":{"entrypoint":"x"},"lifecycle":{"idleTimeoutSeconds":0,"memoryLimitBytes":1}}`, 0},
		"the limit in megabytes": {`{"backend":{"entrypoint":"x"},"lifecycle":{"idleTimeoutSeconds":60,"memoryLimitMB":512}}`, 0},
		"no memory limit":        {`{"backend":{"entrypoint":"x"},"lifecycle":{"idleTimeoutSeconds":60}}`, 1},
		"no idle policy":         {`{"backend":{"entrypoint":"x"},"lifecycle":{"memoryLimitBytes":1}}`, 1},
		"no lifecycle":           {`{"backend":{"entrypoint":"x"}}`, 2},
		"a limit of zero":        {`{"backend":{"entrypoint":"x"},"lifecycle":{"idleTimeoutSeconds":60,"memoryLimitBytes":0}}`, 1},
		"no backend":             {`{"frontend":{"entrypoint":"x"}}`, 0},
	} {
		if got := listLifecycleGaps([]byte(one.manifest)); len(got) != one.gaps {
			t.Errorf("%s: %d gaps found, want %d: %v", name, len(got), one.gaps, got)
		}
	}
}

var backtick = regexp.MustCompile("`([^`]+)`")

// readDocumentedPermissions is the permissions the plugin guide lists: the
// values in backticks before the sentence that rejects any other.
func readDocumentedPermissions(guide string) []string {
	for line := range strings.SplitSeq(guide, "\n") {
		if !strings.HasPrefix(line, "- **`permissions`**") {
			continue
		}
		list, _, _ := strings.Cut(strings.TrimPrefix(line, "- **`permissions`**"), "Any other value")
		var names []string
		for _, match := range backtick.FindAllStringSubmatch(list, -1) {
			names = append(names, match[1])
		}
		return names
	}
	return nil
}

// listPermissionDifferences lists the permissions one side names and the other
// does not.
func listPermissionDifferences(documented, valid []string) []string {
	var differences []string
	for _, name := range documented {
		if !slices.Contains(valid, name) {
			differences = append(differences, "documented but not valid: "+name)
		}
	}
	for _, name := range valid {
		if !slices.Contains(documented, name) {
			differences = append(differences, "valid but not documented: "+name)
		}
	}
	return differences
}

func TestThePermissionsTheGuideListsAreTheValidators(t *testing.T) {
	guide, err := os.ReadFile(filepath.Join(docsRoot, "plugins", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := readDocumentedPermissions(string(guide))
	if len(documented) == 0 {
		t.Fatal("the guide lists no permissions: a rule checked against nothing")
	}
	valid := append(slices.Clone(port.PluginPermissions), port.PermissionNetworkAlias)
	for _, difference := range listPermissionDifferences(documented, valid) {
		t.Error(difference)
	}
	for _, name := range valid {
		if err := (port.PluginManifest{Permissions: []string{name}}).ValidatePermissions(); err != nil {
			t.Errorf("the validator refuses %s: %v", name, err)
		}
	}
}

func TestWhatThePermissionRuleRefuses(t *testing.T) {
	valid := []string{"vault:read", "network:outbound", "network"}
	if got := listPermissionDifferences([]string{"vault:read", "network:outbound", "network"}, valid); len(got) != 0 {
		t.Errorf("equal lists differ: %v", got)
	}
	if got := listPermissionDifferences([]string{"vault:read", "network:outbound"}, valid); len(got) != 1 {
		t.Errorf("a permission left out of the guide is found %d times: %v", len(got), got)
	}
	if got := listPermissionDifferences([]string{"vault:read", "network:outbound", "network", "root"}, valid); len(got) != 1 {
		t.Errorf("a permission only the guide names is found %d times: %v", len(got), got)
	}
	line := "- **`permissions`**: Whitelisted: `vault:read` and `ai:complete` (ask). Any other value, such as `root`, is rejected."
	if got := readDocumentedPermissions("text\n" + line + "\n"); !slices.Equal(got, []string{"vault:read", "ai:complete"}) {
		t.Errorf("the guide's line is read as %v", got)
	}
	if got := readDocumentedPermissions("no such line"); got != nil {
		t.Errorf("a guide without the line is read as %v", got)
	}
}

// listBackendSources is the Go sources a plugin's backend is compiled from: its own,
// to any depth, and those of every directory of this repository that they import.
func listBackendSources(t *testing.T, backend string) []string {
	t.Helper()
	var files []string
	seen := []string{backend}
	queue := []string{backend}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		for _, file := range listGoSources(t, dir, dir == backend) {
			files = append(files, file)
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("%s: %v", file, err)
			}
			for _, spec := range parsed.Imports {
				rest, local := strings.CutPrefix(strings.Trim(spec.Path.Value, `"`), modulesPath)
				imported := filepath.Join(modulesRoot, filepath.FromSlash(rest))
				if local && !slices.Contains(seen, imported) {
					seen = append(seen, imported)
					queue = append(queue, imported)
				}
			}
		}
	}
	return files
}

// listGoSources is the Go files of a directory that are not tests, with those of
// the directories under it where deep is set.
func listGoSources(t *testing.T, dir string, deep bool) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			if path != dir && !deep || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// listFrontendSources is the sources of a plugin's frontend that are not tests.
func listFrontendSources(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			if slices.Contains([]string{"node_modules", "dist"}, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		isTest := strings.Contains(name, ".test.") || strings.Contains(name, ".spec.")
		if slices.Contains([]string{".ts", ".tsx", ".vue", ".js"}, filepath.Ext(name)) && !isTest {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// isNamedIn says whether a source names a key as a string.
func isNamedIn(source, key string) bool {
	for _, quote := range []string{`"`, `'`, "`"} {
		if strings.Contains(source, quote+key+quote) {
			return true
		}
	}
	return false
}

// listUnreadSettings lists the settings a manifest declares that none of the
// sources name. The setting that chooses an agent is read by the host, which
// routes by it.
func listUnreadSettings(manifest port.PluginManifest, sources []string) []string {
	var unread []string
	for _, setting := range manifest.Settings {
		if setting.Type == port.SettingTypeAgent {
			continue
		}
		if !slices.ContainsFunc(sources, func(source string) bool { return isNamedIn(source, setting.Key) }) {
			unread = append(unread, setting.Key)
		}
	}
	return unread
}

func TestEverySettingAPluginDeclaresIsReadByThePlugin(t *testing.T) {
	declared := 0
	for _, one := range getPluginManifests(t) {
		files := listBackendSources(t, filepath.Join(one.Dir, "backend"))
		files = append(files, listFrontendSources(t, filepath.Join(one.Dir, "frontend"))...)
		var sources []string
		for _, file := range files {
			text, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			sources = append(sources, string(text))
		}
		declared += len(one.Settings)
		for _, key := range listUnreadSettings(one.PluginManifest, sources) {
			t.Errorf("%s declares the setting %s and nothing it is compiled from reads it", one.Dir, key)
		}
	}
	if declared == 0 {
		t.Fatal("no plugin declares a setting: a rule checked against nothing")
	}
}

func TestWhatTheSettingRuleRefuses(t *testing.T) {
	manifest := port.PluginManifest{Settings: []port.PluginManifestSetting{
		{Key: "language"}, {Key: "dpi"}, {Key: "agent", Type: port.SettingTypeAgent},
	}}
	if got := listUnreadSettings(manifest, []string{`values["language"]`, "const dpi = 'dpi'"}); len(got) != 0 {
		t.Errorf("settings that are read are found unread: %v", got)
	}
	if got := listUnreadSettings(manifest, []string{`values["language"]`}); !slices.Equal(got, []string{"dpi"}) {
		t.Errorf("a setting nothing names is found as %v", got)
	}
	if got := listUnreadSettings(manifest, []string{`languages := "dpis"`}); !slices.Equal(got, []string{"language", "dpi"}) {
		t.Errorf("a key inside another word is found as read: %v", got)
	}
}
