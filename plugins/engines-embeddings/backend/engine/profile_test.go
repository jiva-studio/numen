package engine

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func getShipped(t *testing.T) *Profiles {
	t.Helper()
	table, err := GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	return table
}

func writeProfile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestShippedProfilesAreValid(t *testing.T) {
	table := getShipped(t)
	for _, id := range table.order {
		if err := table.byID[id].Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestShippedProfilesPinTheFilesOfEveryLocalModel(t *testing.T) {
	table := getShipped(t)
	curated := table.GetCurated()
	if len(curated) < 2 {
		t.Fatalf("GetCurated() returned %d profiles, want at least 2", len(curated))
	}
	for _, p := range curated {
		if p.Tier == "" {
			t.Errorf("%s: unexpected curated profile Tier=%q", p.ID, p.Tier)
		}
		if len(p.Revision) != 40 {
			t.Errorf("%s: revision %q is no commit", p.ID, p.Revision)
		}
		for _, name := range []string{p.OnnxFile, TokenizerFile} {
			if len(p.SHA256[name]) != 64 {
				t.Errorf("%s: no sha256 for %s", p.ID, name)
			}
		}
	}
}

func TestShippedDefaultKeepsItsPrefixes(t *testing.T) {
	p, err := getShipped(t).GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.QueryPrefix != "query: " || p.DocumentPrefix != "passage: " {
		t.Errorf("prefixes = %q, %q; a multilingual-e5 model needs query: and passage:", p.QueryPrefix, p.DocumentPrefix)
	}
	if p.Dimensions != 384 || p.Pooling != PoolMean || p.GetModelFile() != "model_quantized.onnx" {
		t.Errorf("profile = %+v", p)
	}
}

func TestGetLocalAcceptsAnAliasAndAnyCase(t *testing.T) {
	table := getShipped(t)
	for _, name := range []string{"Xenova/bge-m3", "bge-m3", "BGE-M3", "bge-m3-int8", " xenova/bge-m3 "} {
		p, err := table.GetLocal(name)
		if err != nil || p.ID != "Xenova/bge-m3" {
			t.Errorf("GetLocal(%q) = %q, %v", name, p.ID, err)
		}
	}
}

func TestGetLocalOfAnUnknownModelNamesWhereToAddAProfile(t *testing.T) {
	if _, err := getShipped(t).GetLocal("owner/unknown"); err == nil || !strings.Contains(err.Error(), "plugin data directory") {
		t.Errorf("default user folder error = %v", err)
	}
	table := getShipped(t).WithUserProfiles("/data/profiles", func(string, error) {})
	for _, name := range []string{"owner/unknown", "openai/text-embedding-3-small", ""} {
		_, err := table.GetLocal(name)
		if !errors.Is(err, ErrNoProfile) {
			t.Fatalf("GetLocal(%q) error = %v, want ErrNoProfile", name, err)
		}
		if !strings.Contains(err.Error(), "/data/profiles") || !strings.Contains(err.Error(), ".json") {
			t.Errorf("error %q does not name the folder to add a profile to", err)
		}
		if !strings.HasSuffix(err.Error(), "choose one of "+DefaultModel+", onnx-community/harrier-oss-v1-270m-ONNX, Xenova/bge-m3") {
			t.Errorf("error %q does not list local IDs accurately", err)
		}
	}
}

func TestTheShippedTableListsLocalModelsOnly(t *testing.T) {
	table := getShipped(t)
	for _, id := range []string{"text-embedding-3-large", "openai/text-embedding-3-small", "baai/bge-m3", "*"} {
		if _, err := table.GetLocal(id); !errors.Is(err, ErrNoProfile) {
			t.Errorf("GetLocal(%q) = %v, want ErrNoProfile", id, err)
		}
	}
}

func TestAProfileFileCannotMarkAModelAsRemote(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "service.json", `{"id":"owner/service","title":"Service","remote":true,"dims":256,"maxTokens":2048,"pooling":"cls"}`)
	var reported []string
	table := getShipped(t).WithUserProfiles(dir, func(file string, _ error) { reported = append(reported, file) })

	if _, err := table.GetLocal("owner/service"); err == nil || len(reported) != 1 {
		t.Errorf("GetLocal = %v, reported %v, want the file skipped", err, reported)
	}
}

func TestUserProfileAddsAModel(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "mine.json", `{"id":"owner/my-model","title":"Mine","dims":256,"maxTokens":2048,"pooling":"last","queryPrefix":"q: ","onnxFile":"onnx/model.onnx","extraInputs":["position_ids"]}`)
	var reported []string
	table := getShipped(t).WithUserProfiles(dir, func(file string, _ error) { reported = append(reported, file) })

	p, err := table.GetLocal("owner/my-model")
	if err != nil || len(reported) != 0 {
		t.Fatalf("GetLocal = %v, reported %v", err, reported)
	}
	if p.Dimensions != 256 || p.Pooling != PoolLast || p.GetTokens() != MaxChunkTokens || !p.IsNormalised() {
		t.Errorf("profile = %+v", p)
	}
	if !slices.Contains(p.ExtraInputs, InputPositionIDs) {
		t.Errorf("extraInputs = %v", p.ExtraInputs)
	}
}

func TestUserProfileOverridesTheFieldsItNamesAndKeepsTheRest(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "e5.json", `[{"id":"Xenova/multilingual-e5-small","queryPrefix":"search_query: ","documentPrefix":"search_document: "}]`)
	table := getShipped(t).WithUserProfiles(dir, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

	p, err := table.GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.QueryPrefix != "search_query: " || p.DocumentPrefix != "search_document: " {
		t.Errorf("prefixes = %q, %q", p.QueryPrefix, p.DocumentPrefix)
	}
	if p.Dimensions != 384 || p.Revision == "" || len(p.SHA256) == 0 {
		t.Errorf("an unnamed field was lost: %+v", p)
	}
	shipped, err := getShipped(t).GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if shipped.QueryPrefix != "query: " {
		t.Error("the override changed the shipped table")
	}
}

func TestUserProfileThatChangesTheFileDropsThePins(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "e5.json", `{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx"}`)
	table := getShipped(t).WithUserProfiles(dir, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

	p, err := table.GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.OnnxFile != "onnx/model_int8.onnx" || p.Revision != "" || p.SHA256 != nil {
		t.Errorf("pins of the old file kept: %+v", p)
	}

	dirPinned := t.TempDir()
	writeProfile(t, dirPinned, "e5.json", `{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx","sha256":{"onnx/model_int8.onnx":"abc"}}`)
	tablePinned := getShipped(t).WithUserProfiles(dirPinned, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })
	p2, err := tablePinned.GetLocal(DefaultModel)
	if err != nil || p2.SHA256["onnx/model_int8.onnx"] != "abc" || p2.Revision == "" {
		t.Errorf("explicit sha256 pin lost: %+v, %v", p2, err)
	}
}

func TestUserProfileByAnAliasOverridesTheModelOfTheAlias(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "m3.json", `{"id":"bge-m3","floorHint":0.5}`)
	table := getShipped(t).WithUserProfiles(dir, func(string, error) {})

	p, err := table.GetLocal("Xenova/bge-m3")
	if err != nil || p.FloorHint != 0.5 {
		t.Errorf("GetLocal = %+v, %v", p, err)
	}
	if n := len(table.GetCurated()); n != len(getShipped(t).GetCurated()) {
		t.Errorf("an alias added a model: %d curated", n)
	}
}

func TestMalformedUserFilesAreReportedAndSkipped(t *testing.T) {
	dir := t.TempDir()
	writeProfile(t, dir, "a-broken.json", `{"id": "owner/x", `)
	writeProfile(t, dir, "b-nodims.json", `{"id":"owner/y","maxTokens":512,"pooling":"mean","onnxFile":"onnx/m.onnx"}`)
	writeProfile(t, dir, "c-unknown-field.json", `{"id":"owner/z","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","querryPrefix":"x"}`)
	writeProfile(t, dir, "d-half.json", `[{"id":"owner/ok","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"},{"id":"owner/bad","dims":8,"maxTokens":8,"pooling":"median","onnxFile":"m.onnx"}]`)
	writeProfile(t, dir, "e-good.json", `{"id":"owner/good","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"}`)
	writeProfile(t, dir, "f-array.json", `[broken`)
	if err := os.MkdirAll(filepath.Join(dir, "sub.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeProfile(t, dir, "notes.txt", `not a profile`)
	reported := map[string]error{}
	table := getShipped(t).WithUserProfiles(dir, func(file string, err error) { reported[filepath.Base(file)] = err })

	for _, name := range []string{"a-broken.json", "b-nodims.json", "c-unknown-field.json", "d-half.json", "f-array.json"} {
		if reported[name] == nil {
			t.Errorf("%s was not reported", name)
		}
	}
	if len(reported) != 5 {
		t.Errorf("reported %v, want the five broken files", reported)
	}
	for _, id := range []string{"owner/x", "owner/y", "owner/z", "owner/ok", "owner/bad"} {
		if _, err := table.GetLocal(id); err == nil {
			t.Errorf("%s from a skipped file is listed", id)
		}
	}
	if _, err := table.GetLocal("owner/good"); err != nil {
		t.Errorf("a good file next to broken ones was skipped: %v", err)
	}
	if _, err := table.GetLocal(DefaultModel); err != nil {
		t.Errorf("the shipped table lost a model: %v", err)
	}
}

func TestMissingUserFolderIsNoError(t *testing.T) {
	table := getShipped(t).WithUserProfiles(filepath.Join(t.TempDir(), "profiles"), func(file string, err error) {
		t.Errorf("reported %s: %v", file, err)
	})
	if _, err := table.GetLocal(DefaultModel); err != nil {
		t.Error(err)
	}
}

func TestGetTokensCapsALocalModelAtTheChunkLimit(t *testing.T) {
	table := getShipped(t)
	m3, err := table.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	if m3.MaxTokens != 8192 || m3.GetTokens() != MaxChunkTokens {
		t.Errorf("bge-m3 tokens = %d of %d", m3.GetTokens(), m3.MaxTokens)
	}
}

func TestHandlingApplyAddsTheRolePrefixInFrontOfEveryText(t *testing.T) {
	h := Handling{QueryPrefix: "query: ", DocumentPrefix: "passage: "}
	texts := []string{"a", "b"}
	cases := []struct {
		role Role
		want []string
	}{
		{RoleDocument, []string{"passage: a", "passage: b"}},
		{RoleQuery, []string{"query: a", "query: b"}},
	}
	for _, tc := range cases {
		if got := h.Apply(tc.role, texts); !slices.Equal(got, tc.want) {
			t.Errorf("Apply(%v) = %q, want %q", tc.role, got, tc.want)
		}
	}
	if !slices.Equal(texts, []string{"a", "b"}) {
		t.Errorf("Apply changed its input: %q", texts)
	}
	if got := (Handling{}).Apply(RoleQuery, texts); !slices.Equal(got, texts) || &got[0] != &texts[0] {
		t.Errorf("no prefix changed or reallocated the texts: %q", got)
	}
}

func TestEveryCuratedModelWithAQueryPrefixAppliesItToQueries(t *testing.T) {
	for _, p := range getShipped(t).GetCurated() {
		h := p.GetHandling()
		got := h.Apply(RoleQuery, []string{"x"})[0]
		if got != p.QueryPrefix+"x" {
			t.Errorf("%s: a query reads %q without its prefix %q", p.ID, got, p.QueryPrefix)
		}
		if doc := h.Apply(RoleDocument, []string{"x"})[0]; doc != p.DocumentPrefix+"x" {
			t.Errorf("%s: a document reads %q", p.ID, doc)
		}
	}
}

func TestHandlingHashChangesWithWhatItCoversAndWithNothingElse(t *testing.T) {
	base := Handling{QueryPrefix: "query: ", DocumentPrefix: "passage: ", File: "model.onnx", Normalise: true}
	want := base.Hash(512)
	if len(want) != 8 {
		t.Errorf("hash %q is not 8 characters", want)
	}
	same := base
	same.ExtraInputs = []string{InputPositionIDs}
	if got := same.Hash(512); got != want {
		t.Errorf("extra inputs changed the hash: %s", got)
	}
	changed := map[string]string{
		"query prefix":    func() string { h := base; h.QueryPrefix = "q: "; return h.Hash(512) }(),
		"document prefix": func() string { h := base; h.DocumentPrefix = "d: "; return h.Hash(512) }(),
		"file":            func() string { h := base; h.File = "model_q4.onnx"; return h.Hash(512) }(),
		"normalise":       func() string { h := base; h.Normalise = false; return h.Hash(512) }(),
		"max tokens":      base.Hash(256),
	}
	for what, got := range changed {
		if got == want {
			t.Errorf("a change of the %s leaves the hash %s", what, want)
		}
	}
}

func TestHandlingIsZeroOnlyWhenItChangesNothing(t *testing.T) {
	if !(Handling{}).IsZero() {
		t.Error("an empty handling is not zero")
	}
	for _, h := range []Handling{{QueryPrefix: "q"}, {DocumentPrefix: "d"}, {File: "f"}, {Normalise: true}, {ExtraInputs: []string{InputPositionIDs}}} {
		if h.IsZero() {
			t.Errorf("%+v is zero", h)
		}
	}
}

func TestOptionTextsNameTierSizeAndWidth(t *testing.T) {
	p, err := getShipped(t).GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.GetOptionTitle(); got != "E5 small (default, CPU, 118 MB)" {
		t.Errorf("title = %q", got)
	}
	if got := p.GetOptionDetail(); got != "384 dims · Hugging Face" {
		t.Errorf("detail = %q", got)
	}
}

func TestValidateRefusesWhatCannotRun(t *testing.T) {
	ok := Profile{ID: "a/b", Dimensions: 1, MaxTokens: 1, Pooling: PoolMean, OnnxFile: "m.onnx"}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Profile){
		"no id":          func(p *Profile) { p.ID = " " },
		"no dims":        func(p *Profile) { p.Dimensions = 0 },
		"no tokens":      func(p *Profile) { p.MaxTokens = 0 },
		"unknown pool":   func(p *Profile) { p.Pooling = "max" },
		"floor negative": func(p *Profile) { p.FloorHint = -0.1 },
		"floor too high": func(p *Profile) { p.FloorHint = 1 },
		"unknown input":  func(p *Profile) { p.ExtraInputs = []string{"past_key_values"} },
		"no file":        func(p *Profile) { p.OnnxFile = "" },
	} {
		p := ok
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestProfileNormaliseCloneAndErrorPaths(t *testing.T) {
	isTrue, isFalse := true, false
	if !(Profile{Normalise: &isTrue}).IsNormalised() {
		t.Error("expected Normalise=true to be normalised")
	}
	if (Profile{Normalise: &isFalse}).IsNormalised() {
		t.Error("expected Normalise=false not to be normalised")
	}

	if _, err := ReadProfiles([]byte("[broken")); err == nil {
		t.Error("expected ReadProfiles to fail on broken array")
	}
	if _, err := ReadProfiles([]byte(`[{"unknown":1}]`)); err == nil {
		t.Error("expected ReadProfiles to fail on unknown field")
	}

	origEmbedded := embeddedProfiles
	defer func() { embeddedProfiles = origEmbedded }()
	embeddedProfiles = []byte("[broken")
	if _, err := GetShippedProfiles(); err == nil {
		t.Error("expected GetShippedProfiles to fail on broken JSON")
	}
	embeddedProfiles = []byte(`[{"id":"x","dims":0}]`)
	if _, err := GetShippedProfiles(); err == nil {
		t.Error("expected GetShippedProfiles to fail on invalid profile")
	}
	embeddedProfiles = origEmbedded

	table := getShipped(t)
	fileAsDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	reportedDir := false
	table.WithUserProfiles(fileAsDir, func(string, error) { reportedDir = true })
	if !reportedDir {
		t.Error("expected WithUserProfiles to report non-directory path")
	}
	if _, err := table.readUserFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("expected readUserFile to fail on missing file")
	}

	dir := t.TempDir()
	writeProfile(t, dir, "extra.json", `{"id":"Xenova/bge-m3","extraInputs":["position_ids"]}`)
	withExtra := table.WithUserProfiles(dir, func(string, error) {})
	m3, err := withExtra.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	m3.Aliases[0] = "changed"
	m3.ExtraInputs[0] = "changed"
	m3.SHA256[TokenizerFile] = "changed"

	m3Again, err := withExtra.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	if m3Again.Aliases[0] == "changed" || m3Again.ExtraInputs[0] == "changed" || m3Again.SHA256[TokenizerFile] == "changed" {
		t.Errorf("mutating returned Profile mutated table state: %+v", m3Again)
	}
}
