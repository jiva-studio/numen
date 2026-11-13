package container

import (
	"errors"
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

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// modulesRoot is the directory the repository's modules sit in, and docsRoot is
// the directory of the repository's documentation, both from this package.
const (
	modulesRoot = "../../../.."
	docsRoot    = "../../../../docs"
	modulesPath = "github.com/jiva-studio/numen/"
)

// pluginManifest is a plugin's manifest as the tree holds it.
type pluginManifest struct {
	Dir string
	Raw []byte
	manifest.Manifest
}

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
		parsed, err := manifestjson.Read(strings.NewReader(string(raw)))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		manifests = append(manifests, pluginManifest{Dir: filepath.Dir(path), Raw: raw, Manifest: parsed})
	}
	return manifests
}

var backtick = regexp.MustCompile("`([^`]+)`")

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
	grant := manifest.FormatConnectionsUse("<operation>")
	valid := append(manifest.PluginPermissions(), manifest.PermissionNetworkAlias, grant)
	for _, difference := range listPermissionDifferences(documented, valid) {
		t.Error(difference)
	}
	for _, name := range valid {
		if name == grant {
			name = manifest.FormatConnectionsUse("llm.chat")
		}
		if err := (manifest.Manifest{Permissions: []string{name}}).ValidatePermissions(); err != nil {
			t.Errorf("the validator refuses %s: %v", name, err)
		}
	}
}

func TestPermissionRule_ReportsPermissionDifferences(t *testing.T) {
	valid := []string{"vault:read", "link:open", "network"}
	if got := listPermissionDifferences([]string{"vault:read", "link:open", "network"}, valid); len(got) != 0 {
		t.Errorf("equal lists differ: %v", got)
	}
	if got := listPermissionDifferences([]string{"vault:read", "link:open"}, valid); len(got) != 1 {
		t.Errorf("a permission left out of the guide is found %d times: %v", len(got), got)
	}
	if got := listPermissionDifferences([]string{"vault:read", "link:open", "network", "root"}, valid); len(got) != 1 {
		t.Errorf("a permission only the guide names is found %d times: %v", len(got), got)
	}
	line := "- **`permissions`**: Whitelisted: `vault:read` and `index:read`. Any other value, such as `root`, is rejected."
	if got := readDocumentedPermissions("text\n" + line + "\n"); !slices.Equal(got, []string{"vault:read", "index:read"}) {
		t.Errorf("the guide's line is read as %v", got)
	}
	if got := readDocumentedPermissions("no such line"); got != nil {
		t.Errorf("a guide without the line is read as %v", got)
	}
}

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
	valid := manifest.PluginCapabilities()
	for _, difference := range listDifferences(documented, valid, "capability") {
		t.Error(difference)
	}
	for _, name := range valid {
		if hasIssue(manifest.Manifest{Capabilities: []string{name}}, manifest.ErrInvalidCapability) {
			t.Errorf("the validator refuses %s", name)
		}
	}
	if !hasIssue(manifest.Manifest{Capabilities: []string{"not-a-capability"}}, manifest.ErrInvalidCapability) {
		t.Error("the validator accepts a capability no host resolves")
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
	valid := manifest.PluginActivationEvents()
	for _, difference := range listDifferences(documented, valid, "activation event") {
		t.Error(difference)
	}
	for _, name := range valid {
		if hasIssue(manifest.Manifest{ActivationEvents: []string{name}}, manifest.ErrInvalidActivationEvent) {
			t.Errorf("the validator refuses %s", name)
		}
	}
	for _, invalid := range []string{"*", "onStartup", "onFile:pdf", "onCommand:run"} {
		if !hasIssue(manifest.Manifest{ActivationEvents: []string{invalid}}, manifest.ErrInvalidActivationEvent) {
			t.Errorf("the validator accepts %s", invalid)
		}
	}
}

func TestShippedPluginsDeclareNoWildcardActivationEvent(t *testing.T) {
	byID := map[string]manifest.Manifest{}
	for _, one := range getPluginManifests(t) {
		if slices.Contains(one.ActivationEvents, "*") {
			t.Errorf("%s declares the wildcard activation event", one.ID)
		}
		byID[one.ID] = one.Manifest
	}
	for _, id := range []string{"md.numen.engines.embeddings", "md.numen.agents.mcp.server"} {
		m, ok := byID[id]
		if !ok {
			t.Fatalf("missing plugin %s", id)
		}
		if len(m.ActivationEvents) != 0 {
			t.Errorf("%s declares activationEvents %v, want none", id, m.ActivationEvents)
		}
	}
	relevance, ok := byID["md.numen.tools.search.relevance"]
	if !ok {
		t.Fatal("missing plugin md.numen.tools.search.relevance")
	}
	if !slices.Equal(relevance.ActivationEvents, []string{manifest.ActivationEventSearchSourceReady}) {
		t.Errorf("md.numen.tools.search.relevance activationEvents = %v, want [onSearchSourceReady]", relevance.ActivationEvents)
	}
}

func hasIssue(m manifest.Manifest, rule error) bool {
	return slices.ContainsFunc(m.ListIssues(), func(issue error) bool { return errors.Is(issue, rule) })
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
	eventGuide := "- **`activationEvents`**: Triggers:\n  - `onSearchSourceReady` (when the search source has indexed a vault)\n- **`next`**:\n"
	if got := readDocumentedActivationEvents(eventGuide); !slices.Equal(got, []string{"onSearchSourceReady"}) {
		t.Errorf("the activation events section is read as %v", got)
	}
}

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

func isNamedIn(source, key string) bool {
	for _, quote := range []string{`"`, `'`, "`"} {
		if strings.Contains(source, quote+key+quote) {
			return true
		}
	}
	return false
}

func isReadByCondition(manifest manifest.Manifest, key string) bool {
	conditions := make([]string, 0, len(manifest.Contributes.Triggers)+len(manifest.Contributes.Handlers))
	for _, one := range manifest.Contributes.Triggers {
		conditions = append(conditions, one.When)
	}
	for _, one := range manifest.Contributes.Handlers {
		conditions = append(conditions, one.When)
	}
	name := "config." + manifest.ID + "." + key
	return slices.ContainsFunc(conditions, func(condition string) bool { return strings.Contains(condition, name) })
}

func listUnreadSettings(declaring manifest.Manifest, sources []string) []string {
	var unread []string
	for _, setting := range declaring.Settings {
		if setting.Type == manifest.SettingTypeConnections {
			continue
		}
		if isReadByCondition(declaring, setting.Key) {
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
		for _, key := range listUnreadSettings(one.Manifest, sources) {
			t.Errorf("%s declares the setting %s and nothing it is compiled from reads it", one.Dir, key)
		}
	}
	if declared == 0 {
		t.Fatal("no plugin declares a setting: a rule checked against nothing")
	}
}

func TestSettingRule_ReportsUnreadSettings(t *testing.T) {
	scans := manifest.Manifest{Settings: []manifest.Setting{
		{Key: "language"}, {Key: "dpi"},
	}}
	if got := listUnreadSettings(scans, []string{`values["language"]`, "const dpi = 'dpi'"}); len(got) != 0 {
		t.Errorf("settings that are read are found unread: %v", got)
	}
	if got := listUnreadSettings(scans, []string{`values["language"]`}); !slices.Equal(got, []string{"dpi"}) {
		t.Errorf("a setting nothing names is found as %v", got)
	}
	if got := listUnreadSettings(scans, []string{`languages := "dpis"`}); !slices.Equal(got, []string{"language", "dpi"}) {
		t.Errorf("a key inside another word is found as read: %v", got)
	}
	rows := manifest.Manifest{Settings: []manifest.Setting{{Key: "connections", Type: manifest.SettingTypeConnections}}}
	if got := listUnreadSettings(rows, nil); len(got) != 0 {
		t.Errorf("a setting of rows, which the host reads, is found unread: %v", got)
	}
	triggered := manifest.Manifest{ID: "org.example.scans", Settings: []manifest.Setting{{Key: "isAutomatic"}}}
	triggered.Contributes.Triggers = []manifest.Trigger{{Event: "file.added", Command: "app:file.createText", When: "config.org.example.scans.isAutomatic == true"}}
	if got := listUnreadSettings(triggered, nil); len(got) != 0 {
		t.Errorf("a setting the condition of a trigger reads is found unread: %v", got)
	}
}

// The capabilities a manifest may declare are those the adapters of the core resolve a plugin by.
func TestTheCapabilitiesAManifestMayDeclareAreTheOnesTheHostResolves(t *testing.T) {
	want := []string{bridge.CapabilitySearchSource, bridge.CapabilityTools}
	got := slices.Sorted(slices.Values(manifest.PluginCapabilities()))
	if !slices.Equal(got, want) {
		t.Errorf("PluginCapabilities = %v, want %v", got, want)
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

func listValidPluginKinds() []string {
	return []string{
		"agents",
		"readers",
		"editors",
		"views",
		"engines",
		"connections",
		"tools",
		"themes",
	}
}

func listDomainPlugins() []string {
	return []string{"notes"}
}

func getKebabName(id string) string {
	var out strings.Builder
	for _, r := range id {
		switch {
		case r == '.':
			out.WriteRune('-')
		case r >= 'A' && r <= 'Z':
			out.WriteRune('-')
			out.WriteRune(r - 'A' + 'a')
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}

func listPluginNamingIssues(folder, id string) []string {
	var issues []string
	if !strings.HasPrefix(id, "md.numen.") {
		issues = append(issues, fmt.Sprintf("id %q does not start with md.numen.", id))
	}
	dotFolder := strings.ReplaceAll(folder, "-", ".")
	name, hasPrefix := strings.CutPrefix(id, "md.numen.")
	if !hasPrefix || getKebabName(name) != strings.ReplaceAll(folder, ".", "-") {
		issues = append(issues, fmt.Sprintf("folder %q differs from id %q without md.numen.", folder, id))
	}
	parts := strings.Split(dotFolder, ".")
	if slices.Contains(listDomainPlugins(), dotFolder) {
		return issues
	}
	if len(parts) < 2 || !slices.Contains(listValidPluginKinds(), parts[0]) {
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

func TestPluginNamingRule_ReportsNamingIssues(t *testing.T) {
	for name, one := range map[string]struct {
		folder string
		id     string
		issues int
	}{
		"valid kebab plugin":      {"views-file-tree", "md.numen.views.file.tree", 0},
		"valid dot plugin":        {"views.file.tree", "md.numen.views.file.tree", 0},
		"folder differs from id":  {"views-file-tree", "md.numen.views.other", 1},
		"unknown kind":            {"widgets-tree", "md.numen.widgets.tree", 1},
		"missing md.numen prefix": {"views-file-tree", "views.file.tree", 2},
		"no name segment":         {"views", "md.numen.views", 1},
		"domain plugin":           {"notes", "md.numen.notes", 0},
		"camelCase segment":       {"views-title-bar", "md.numen.views.titleBar", 0},
		"camelCase folder":        {"views-titleBar", "md.numen.views.titleBar", 1},
	} {
		if got := listPluginNamingIssues(one.folder, one.id); len(got) != one.issues {
			t.Errorf("%s: %d issues found, want %d: %v", name, len(got), one.issues, got)
		}
	}
}
