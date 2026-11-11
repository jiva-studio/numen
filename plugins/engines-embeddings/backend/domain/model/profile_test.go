package model

import (
	"errors"
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
		if p.Remote || p.Tier == "" {
			t.Errorf("%s: unexpected curated profile Remote=%v Tier=%q", p.ID, p.Remote, p.Tier)
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
	table := getShipped(t).WithUserProfiles("/data/profiles", nil, func(string, error) {})
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

func TestGetRemoteOfAnUnlistedModelUsesTheFallbackWithItsOwnID(t *testing.T) {
	table := getShipped(t)
	shipped := table.GetRemote("text-embedding-3-large")
	if shipped.Dimensions != 3072 {
		t.Errorf("dims = %d, want 3072", shipped.Dimensions)
	}
	if fromLocal := table.GetRemote(DefaultModel); !fromLocal.Remote || fromLocal.Dimensions != 1024 {
		t.Errorf("GetRemote(DefaultModel) = %+v, want fallback remote profile", fromLocal)
	}
	other := table.GetRemote("vendor/some-embedder")
	if other.ID != "vendor/some-embedder" || other.Dimensions != 1024 || !other.Remote {
		t.Errorf("fallback = %+v", other)
	}
	if !other.GetHandling().IsZero() {
		t.Errorf("a remote model has handling %+v, want none", other.GetHandling())
	}
}

func TestUserProfileAddsAModel(t *testing.T) {
	var reported []string
	table := getShipped(t).WithUserProfiles("/data/profiles", []UserFile{{Path: "mine.json", Data: []byte(`{"id":"owner/my-model","title":"Mine","dims":256,"maxTokens":2048,"pooling":"last","queryPrefix":"q: ","onnxFile":"onnx/model.onnx","extraInputs":["position_ids"]}`)}}, func(file string, _ error) { reported = append(reported, file) })

	p, err := table.GetLocal("owner/my-model")
	if err != nil || len(reported) != 0 {
		t.Fatalf("GetLocal = %v, reported %v", err, reported)
	}
	if p.Dimensions != 256 || p.Pooling != PoolLast || p.GetTokens() != MaxChunkTokens || !p.IsNormalized() {
		t.Errorf("profile = %+v", p)
	}
	if !slices.Contains(p.ExtraInputs, InputPositionIDs) {
		t.Errorf("extraInputs = %v", p.ExtraInputs)
	}
}

func TestUserProfileOverridesTheFieldsItNamesAndKeepsTheRest(t *testing.T) {
	table := getShipped(t).WithUserProfiles("/data/profiles", []UserFile{{Path: "e5.json", Data: []byte(`[{"id":"Xenova/multilingual-e5-small","queryPrefix":"search_query: ","documentPrefix":"search_document: "}]`)}}, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

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
	table := getShipped(t).WithUserProfiles("/data/profiles", []UserFile{{Path: "e5.json", Data: []byte(`{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx"}`)}}, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })

	p, err := table.GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.OnnxFile != "onnx/model_int8.onnx" || p.Revision != "" || p.SHA256 != nil {
		t.Errorf("pins of the old file kept: %+v", p)
	}

	tablePinned := getShipped(t).WithUserProfiles("/data/profiles", []UserFile{{Path: "e5.json", Data: []byte(`{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx","sha256":{"onnx/model_int8.onnx":"abc"}}`)}}, func(file string, err error) { t.Errorf("reported %s: %v", file, err) })
	p2, err := tablePinned.GetLocal(DefaultModel)
	if err != nil || p2.SHA256["onnx/model_int8.onnx"] != "abc" || p2.Revision == "" {
		t.Errorf("explicit sha256 pin lost: %+v, %v", p2, err)
	}
}

func TestUserProfileByAnAliasOverridesTheModelOfTheAlias(t *testing.T) {
	table := getShipped(t).WithUserProfiles("/data/profiles", []UserFile{{Path: "m3.json", Data: []byte(`{"id":"bge-m3","floorHint":0.5}`)}}, func(string, error) {})

	p, err := table.GetLocal("Xenova/bge-m3")
	if err != nil || p.FloorHint != 0.5 {
		t.Errorf("GetLocal = %+v, %v", p, err)
	}
	if n := len(table.GetCurated()); n != len(getShipped(t).GetCurated()) {
		t.Errorf("an alias added a model: %d curated", n)
	}
}

func TestMalformedUserFilesAreReportedAndSkipped(t *testing.T) {
	files := []UserFile{}
	files = append(files, UserFile{Path: "a-broken.json", Data: []byte(`{"id": "owner/x", `)})
	files = append(files, UserFile{Path: "b-nodims.json", Data: []byte(`{"id":"owner/y","maxTokens":512,"pooling":"mean","onnxFile":"onnx/m.onnx"}`)})
	files = append(files, UserFile{Path: "c-unknown-field.json", Data: []byte(`{"id":"owner/z","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","querryPrefix":"x"}`)})
	files = append(files, UserFile{Path: "d-half.json", Data: []byte(`[{"id":"owner/ok","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"},{"id":"owner/bad","dims":8,"maxTokens":8,"pooling":"median","onnxFile":"m.onnx"}]`)})
	files = append(files, UserFile{Path: "e-good.json", Data: []byte(`{"id":"owner/good","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"}`)})
	files = append(files, UserFile{Path: "f-array.json", Data: []byte(`[broken`)})
	reported := map[string]error{}
	table := getShipped(t).WithUserProfiles("/data/profiles", files, func(file string, err error) { reported[file] = err })

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

func TestGetTokensCapsALocalModelAtTheChunkLimit(t *testing.T) {
	table := getShipped(t)
	m3, err := table.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	if m3.MaxTokens != 8192 || m3.GetTokens() != MaxChunkTokens {
		t.Errorf("bge-m3 tokens = %d of %d", m3.GetTokens(), m3.MaxTokens)
	}
	if got := table.GetRemote("text-embedding-3-large").GetTokens(); got != 8191 {
		t.Errorf("a remote model is capped at %d", got)
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
	base := Handling{QueryPrefix: "query: ", DocumentPrefix: "passage: ", File: "model.onnx", Normalize: true}
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
		"normalise":       func() string { h := base; h.Normalize = false; return h.Hash(512) }(),
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
	for _, h := range []Handling{{QueryPrefix: "q"}, {DocumentPrefix: "d"}, {File: "f"}, {Normalize: true}, {ExtraInputs: []string{InputPositionIDs}}} {
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

func TestProfileNormalizeCloneAndErrorPaths(t *testing.T) {
	isTrue, isFalse := true, false
	if !(Profile{Normalize: &isTrue}).IsNormalized() {
		t.Error("expected Normalize=true to be normalized")
	}
	if (Profile{Normalize: &isFalse}).IsNormalized() {
		t.Error("expected Normalize=false not to be normalized")
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
	withExtra := table.WithUserProfiles("/data/profiles", []UserFile{{Path: "extra.json", Data: []byte(`{"id":"Xenova/bge-m3","extraInputs":["position_ids"]}`)}}, func(string, error) {})
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
