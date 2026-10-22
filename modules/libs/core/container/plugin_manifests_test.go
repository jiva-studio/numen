package container

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
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
	pluginport.PluginManifest
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
		parsed, err := pluginport.ReadManifest(strings.NewReader(string(raw)))
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
	valid := append(pluginport.PluginPermissions(), pluginport.PermissionNetworkAlias)
	for _, difference := range listPermissionDifferences(documented, valid) {
		t.Error(difference)
	}
	for _, name := range valid {
		if err := (pluginport.PluginManifest{Permissions: []string{name}}).ValidatePermissions(); err != nil {
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

// readDocumentedCapabilities is the capabilities the plugin guide lists: the
// values in backticks on the capabilities line.
func readDocumentedCapabilities(guide string) []string {
	for line := range strings.SplitSeq(guide, "\n") {
		if !strings.HasPrefix(line, "- **`capabilities`**") {
			continue
		}
		rest := strings.TrimPrefix(line, "- **`capabilities`**")
		var names []string
		for _, match := range backtick.FindAllStringSubmatch(rest, -1) {
			names = append(names, match[1])
		}
		return names
	}
	return nil
}

// readDocumentedActivationEvents is the activation event patterns the plugin guide lists:
// the values in backticks on the bullet lines of the activationEvents section.
func readDocumentedActivationEvents(guide string) []string {
	var inSection bool
	var events []string
	for line := range strings.SplitSeq(guide, "\n") {
		if strings.HasPrefix(line, "- **`activationEvents`**") {
			inSection = true
			continue
		}
		if inSection {
			if strings.HasPrefix(line, "  - `") {
				if match := backtick.FindStringSubmatch(line); len(match) > 1 {
					events = append(events, match[1])
				}
			} else if strings.HasPrefix(line, "- **`") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "## ") {
				break
			}
		}
	}
	return events
}

// listDifferences lists the items one side names and the other does not.
func listDifferences(documented, valid []string, itemKind string) []string {
	var differences []string
	for _, name := range documented {
		if !slices.Contains(valid, name) {
			differences = append(differences, fmt.Sprintf("%s documented but not valid: %s", itemKind, name))
		}
	}
	for _, name := range valid {
		if !slices.Contains(documented, name) {
			differences = append(differences, fmt.Sprintf("%s valid but not documented: %s", itemKind, name))
		}
	}
	return differences
}

func TestTheCapabilitiesTheGuideListsAreTheValidators(t *testing.T) {
	guide, err := os.ReadFile(filepath.Join(docsRoot, "plugins", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := readDocumentedCapabilities(string(guide))
	if len(documented) == 0 {
		t.Fatal("the guide lists no capabilities: a rule checked against nothing")
	}
	valid := pluginport.PluginCapabilities()
	for _, difference := range listDifferences(documented, valid, "capability") {
		t.Error(difference)
	}
	for _, name := range valid {
		if err := (pluginport.PluginManifest{Capabilities: []string{name}}).ValidateCapabilities(); err != nil {
			t.Errorf("the validator refuses %s: %v", name, err)
		}
	}
}

func TestTheActivationEventsTheGuideListsAreTheValidators(t *testing.T) {
	guide, err := os.ReadFile(filepath.Join(docsRoot, "plugins", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	documented := readDocumentedActivationEvents(string(guide))
	if len(documented) == 0 {
		t.Fatal("the guide lists no activation events: a rule checked against nothing")
	}
	valid := pluginport.PluginActivationEvents()
	for _, difference := range listDifferences(documented, valid, "activation event") {
		t.Error(difference)
	}
	for _, name := range valid {
		if err := (pluginport.PluginManifest{ActivationEvents: []string{name}}).ValidateActivationEvents(); err != nil {
			t.Errorf("the validator refuses %s: %v", name, err)
		}
	}
}

func TestWhatTheCapabilityAndEventDocumentationRulesRefuse(t *testing.T) {
	valid := []string{"document", "book"}
	if got := listDifferences([]string{"document", "book"}, valid, "capability"); len(got) != 0 {
		t.Errorf("equal lists differ: %v", got)
	}
	if got := listDifferences([]string{"document"}, valid, "capability"); len(got) != 1 {
		t.Errorf("a capability left out of the guide is found %d times: %v", len(got), got)
	}
	if got := listDifferences([]string{"document", "book", "other"}, valid, "capability"); len(got) != 1 {
		t.Errorf("a capability only the guide names is found %d times: %v", len(got), got)
	}
	guideLine := "- **`capabilities`**: Core capabilities provided (`document`, `book`)."
	if got := readDocumentedCapabilities(guideLine); !slices.Equal(got, []string{"document", "book"}) {
		t.Errorf("the capabilities line is read as %v", got)
	}
	eventGuide := "- **`activationEvents`**: Triggers:\n  - `onFile:<glob>` (e.g. pdf)\n  - `*` (startup)\n- **`next`**:\n"
	if got := readDocumentedActivationEvents(eventGuide); !slices.Equal(got, []string{"onFile:<glob>", "*"}) {
		t.Errorf("the activation events section is read as %v", got)
	}
}

// listBackendSources is the Go sources a plugin's backend is compiled from: its own,
// to any depth, and those of every directory of this repository that they import.
func listBackendSources(t *testing.T, backend string) []string {
	t.Helper()
	var files []string
	observed := []string{backend}
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
				if local && !slices.Contains(observed, imported) {
					observed = append(observed, imported)
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
func listUnreadSettings(manifest pluginport.PluginManifest, sources []string) []string {
	var unread []string
	for _, setting := range manifest.Settings {
		if setting.Type == pluginport.SettingTypeAgent {
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
	manifest := pluginport.PluginManifest{Settings: []pluginport.PluginManifestSetting{
		{Key: "language"}, {Key: "dpi"}, {Key: "agent", Type: pluginport.SettingTypeAgent},
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

// coreCapabilities are the capabilities resolved by the core's adapters and composition root.
var coreCapabilities = map[string]bool{
	plugin.CapabilityDocument:      true,
	plugin.CapabilityBook:          true,
	plugin.CapabilityRecognition:   true,
	plugin.CapabilityTranscription: true,
	plugin.CapabilityEmbedding:     true,
	plugin.CapabilityDownloader:    true,
	plugin.CapabilityProofreading:  true,
	plugin.CapabilityTools:         true,
	"ai.agent.provider":            true,
}

// sdkCapabilities are the capabilities resolved by the frontend SDK and tab kinds.
var sdkCapabilities = map[string]bool{
	"audio":     true,
	"recording": true,
	"media":     true,
	"note":      true,
}

// capabilityMarkers are explicit capability markers declared by plugins.
var capabilityMarkers = map[string]bool{
	"agent": true,
}

func isResolvedCapability(capability string) bool {
	return coreCapabilities[capability] || sdkCapabilities[capability] || capabilityMarkers[capability]
}

func TestEveryCapabilityInAManifestIsResolvedOrMarked(t *testing.T) {
	manifests := getPluginManifests(t)
	declared := 0
	for _, one := range manifests {
		for _, capability := range one.Capabilities {
			declared++
			if !isResolvedCapability(capability) {
				t.Errorf("%s declares capability %q which is neither resolved nor an explicit marker", one.Dir, capability)
			}
		}
	}
	if declared == 0 {
		t.Fatal("no plugin declares a capability: a rule checked against nothing")
	}
}

func TestWhatTheCapabilityRuleRefuses(t *testing.T) {
	var violations []string
	for _, capability := range []string{
		"document",
		"audio",
		"views.agent.chat",
		"unknown.capability",
		"unresolved",
	} {
		if !isResolvedCapability(capability) {
			violations = append(violations, capability)
		}
	}
	want := []string{"views.agent.chat", "unknown.capability", "unresolved"}
	if !slices.Equal(violations, want) {
		t.Errorf("the capability rule rejects %v, want %v", violations, want)
	}
}

// isEmittedActivationEvent answers whether an activation event is one the host emits.
func isEmittedActivationEvent(event string) bool {
	switch {
	case event == "*":
		return true
	case strings.HasPrefix(event, "onFile:"):
		return len(strings.TrimPrefix(event, "onFile:")) > 0
	case strings.HasPrefix(event, "onCommand:"):
		return len(strings.TrimPrefix(event, "onCommand:")) > 0
	default:
		return false
	}
}

func TestEveryActivationEventIsOneTheHostEmits(t *testing.T) {
	manifests := getPluginManifests(t)
	declared := 0
	for _, one := range manifests {
		for _, event := range one.ActivationEvents {
			declared++
			if !isEmittedActivationEvent(event) {
				t.Errorf("%s declares activation event %q which is not emitted by the host", one.Dir, event)
			}
		}
	}
	if declared == 0 {
		t.Fatal("no plugin declares an activation event: a rule checked against nothing")
	}
}

func TestWhatTheActivationEventRuleRefuses(t *testing.T) {
	var violations []string
	for _, event := range []string{
		"*",
		"onFile:*.pdf",
		"onCommand:ocr.recognize",
		"onStartupFinished",
		"onView:agent",
		"unsupportedEvent",
		"onFile:",
		"onCommand:",
		"onView:",
	} {
		if !isEmittedActivationEvent(event) {
			violations = append(violations, event)
		}
	}
	want := []string{"onStartupFinished", "onView:agent", "unsupportedEvent", "onFile:", "onCommand:", "onView:"}
	if !slices.Equal(violations, want) {
		t.Errorf("the activation event rule rejects %v, want %v", violations, want)
	}
}

func TestEveryPluginHasAReadme(t *testing.T) {
	for _, one := range getPluginManifests(t) {
		readme := filepath.Join(one.Dir, "README.md")
		if _, err := os.Stat(readme); err != nil {
			t.Errorf("%s has no README.md: %v", one.Dir, err)
		}
	}
}

// validPluginKinds is every plugin kind known to numen.
var validPluginKinds = []string{
	"agents",
	"readers",
	"editors",
	"views",
	"engines",
	"tools",
}

// listPluginNamingIssues lists what a plugin folder and id break: the id starts
// id, and the kind is one of agents, readers, editors, views, engines or tools.
func listPluginNamingIssues(folder, id string) []string {
	var issues []string
	if !strings.HasPrefix(id, "md.numen.") {
		issues = append(issues, fmt.Sprintf("id %q does not start with md.numen.", id))
	}
	if strings.Contains(id, "-") {
		issues = append(issues, fmt.Sprintf("id %q contains a hyphen", id))
	}
	expectedID := "md.numen." + folder
	if id != expectedID {
		issues = append(issues, fmt.Sprintf("folder %q differs from id %q without md.numen.", folder, id))
	}
	parts := strings.Split(folder, ".")
	if len(parts) < 2 || !slices.Contains(validPluginKinds, parts[0]) {
		issues = append(issues, fmt.Sprintf("kind %q in folder %q is not a known plugin kind", parts[0], folder))
	}
	return issues
}

func TestEveryPluginFollowsTheNamingRule(t *testing.T) {
	for _, one := range getPluginManifests(t) {
		folder := filepath.Base(one.Dir)
		for _, issue := range listPluginNamingIssues(folder, one.ID) {
			t.Errorf("%s: %s", one.Dir, issue)
		}
	}
}

func TestWhatThePluginNamingRuleRefuses(t *testing.T) {
	for name, one := range map[string]struct {
		folder string
		id     string
		issues int
	}{
		"valid plugin":            {"views.file.tree", "md.numen.views.file.tree", 0},
		"folder differs from id":  {"views.file.tree", "md.numen.views.other", 1},
		"id contains hyphen":      {"views.file-tree", "md.numen.views.file-tree", 1},
		"unknown kind":            {"widgets.tree", "md.numen.widgets.tree", 1},
		"missing md.numen prefix": {"views.file.tree", "views.file.tree", 2},
		"no name segment":         {"views", "md.numen.views", 1},
	} {
		if got := listPluginNamingIssues(one.folder, one.id); len(got) != one.issues {
			t.Errorf("%s: %d issues found, want %d: %v", name, len(got), one.issues, got)
		}
	}
}
