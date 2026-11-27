package model

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func getShippedProfiles(t *testing.T) *Profiles {
	t.Helper()
	table, err := GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	return table
}

func TestShippedProfilesAreValid(t *testing.T) {
	table := getShippedProfiles(t)
	for _, id := range table.order {
		if err := table.byID[id].Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestShippedProfilesPinTheFilesOfEveryLocalModel(t *testing.T) {
	table := getShippedProfiles(t)
	curated := table.GetCurated()
	if len(curated) < 2 {
		t.Fatalf("GetCurated() returned %d profiles, want at least 2", len(curated))
	}
	for _, p := range curated {
		if p.Tier == "" {
			t.Errorf("%s: unexpected curated profile Tier=%q", p.ID, p.Tier)
		}
		if p.Revision != "" || p.SHA256 != nil {
			t.Errorf("%s: a shipped profile pins its files by address: %q %v", p.ID, p.Revision, p.SHA256)
		}
		for _, name := range []string{p.OnnxFile, TokenizerFile} {
			if p.Dependencies[name] == "" {
				t.Errorf("%s: no dependency for %s", p.ID, name)
			}
		}
	}
}

func TestShippedDefaultKeepsItsPrefixes(t *testing.T) {
	p, err := getShippedProfiles(t).GetLocal(DefaultModel)
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
	table := getShippedProfiles(t)
	for _, name := range []string{"Xenova/bge-m3", "bge-m3", "BGE-M3", "bge-m3-int8", " xenova/bge-m3 "} {
		p, err := table.GetLocal(name)
		if err != nil || p.ID != "Xenova/bge-m3" {
			t.Errorf("GetLocal(%q) = %q, %v", name, p.ID, err)
		}
	}
}

func TestGetLocalOfAnUnknownModelNamesWhereToAddAProfile(t *testing.T) {
	if _, err := getShippedProfiles(t).GetLocal("owner/unknown"); err == nil || !strings.Contains(err.Error(), "plugin data directory") {
		t.Errorf("default user folder error = %v", err)
	}
	table := getShippedProfiles(t).WithUserProfiles("/data/profiles", nil, func(string, error) {})
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
	table := getShippedProfiles(t)
	for _, id := range []string{"text-embedding-3-large", "openai/text-embedding-3-small", "baai/bge-m3", "*"} {
		if _, err := table.GetLocal(id); !errors.Is(err, ErrNoProfile) {
			t.Errorf("GetLocal(%q) = %v, want ErrNoProfile", id, err)
		}
	}
}

func TestAProfileFileCannotMarkAModelAsRemote(t *testing.T) {
	var reported []string
	files := []UserFile{{Path: "service.json", Data: []byte(`{"id":"owner/service","title":"Service","remote":true,"dims":256,"maxTokens":2048,"pooling":"cls"}`)}}
	table := getShippedProfiles(t).WithUserProfiles("/data/profiles", files, func(file string, _ error) { reported = append(reported, file) })

	if _, err := table.GetLocal("owner/service"); err == nil || len(reported) != 1 {
		t.Errorf("GetLocal = %v, reported %v, want the file skipped", err, reported)
	}
}

func TestUserProfileAddsAModel(t *testing.T) {
	var reported []string
	table := getShippedProfiles(
		t,
	).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "mine.json", Data: []byte(
			`{"id":"owner/my-model","title":"Mine","dims":256,"maxTokens":2048,` +
				`"pooling":"last","queryPrefix":"q: ","onnxFile":"onnx/model.onnx","tokenizer":"bpe","extraInputs":["position_ids"]}`,
		)}},
		func(file string, _ error) { reported = append(reported, file) },
	)

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
	table := getShippedProfiles(
		t,
	).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "e5.json", Data: []byte(`[{"id":"Xenova/multilingual-e5-small","queryPrefix":"search_query: ","documentPrefix":"search_document: "}]`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)

	p, err := table.GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.QueryPrefix != "search_query: " || p.DocumentPrefix != "search_document: " {
		t.Errorf("prefixes = %q, %q", p.QueryPrefix, p.DocumentPrefix)
	}
	if p.Dimensions != 384 || len(p.Dependencies) == 0 {
		t.Errorf("an unnamed field was lost: %+v", p)
	}
	shipped, err := getShippedProfiles(t).GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if shipped.QueryPrefix != "query: " {
		t.Error("the override changed the shipped table")
	}
}

func TestUserProfileThatChangesTheFileDropsThePins(t *testing.T) {
	table := getShippedProfiles(
		t,
	).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "e5.json", Data: []byte(`{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx"}`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)

	p, err := table.GetLocal(DefaultModel)
	if err != nil {
		t.Fatal(err)
	}
	if p.OnnxFile != "onnx/model_int8.onnx" || p.Revision != "" || p.SHA256 != nil || p.Dependencies != nil {
		t.Errorf("pins of the old file kept: %+v", p)
	}

	tablePinned := getShippedProfiles(
		t,
	).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "e5.json", Data: []byte(`{"id":"Xenova/multilingual-e5-small","onnxFile":"onnx/model_int8.onnx","sha256":{"onnx/model_int8.onnx":"abc"}}`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)
	p2, err := tablePinned.GetLocal(DefaultModel)
	if err != nil || p2.SHA256["onnx/model_int8.onnx"] != "abc" || p2.Dependencies != nil {
		t.Errorf("explicit sha256 pin lost: %+v, %v", p2, err)
	}
}

func TestUserProfileByAnAliasOverridesTheModelOfTheAlias(t *testing.T) {
	table := getShippedProfiles(
		t,
	).WithUserProfiles("/data/profiles", []UserFile{{Path: "m3.json", Data: []byte(`{"id":"bge-m3","floorHint":0.5}`)}}, func(string, error) {})

	p, err := table.GetLocal("Xenova/bge-m3")
	if err != nil || p.FloorHint != 0.5 {
		t.Errorf("GetLocal = %+v, %v", p, err)
	}
	if n := len(table.GetCurated()); n != len(getShippedProfiles(t).GetCurated()) {
		t.Errorf("an alias added a model: %d curated", n)
	}
}

func TestMalformedUserFilesAreReportedAndSkipped(t *testing.T) {
	files := []UserFile{}
	files = append(files, UserFile{Path: "a-broken.json", Data: []byte(`{"id": "owner/x", `)})
	files = append(files, UserFile{Path: "b-nodims.json", Data: []byte(`{"id":"owner/y","maxTokens":512,"pooling":"mean","onnxFile":"onnx/m.onnx","tokenizer":"bpe"}`)})
	files = append(
		files,
		UserFile{Path: "c-unknown-field.json", Data: []byte(`{"id":"owner/z","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","tokenizer":"bpe","querryPrefix":"x"}`)},
	)
	files = append(
		files,
		UserFile{
			Path: "d-half.json",
			Data: []byte(
				`[{"id":"owner/ok","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","tokenizer":"bpe"},{"id":"owner/bad","dims":8,"maxTokens":8,"pooling":"median","onnxFile":"m.onnx","tokenizer":"bpe"}]`,
			),
		},
	)
	files = append(files, UserFile{Path: "e-good.json", Data: []byte(`{"id":"owner/good","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","tokenizer":"bpe"}`)})
	files = append(files, UserFile{Path: "f-array.json", Data: []byte(`[broken`)})
	reported := map[string]error{}
	table := getShippedProfiles(t).WithUserProfiles("/data/profiles", files, func(file string, err error) { reported[file] = err })

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
	table := getShippedProfiles(t)
	m3, err := table.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	if m3.MaxTokens != 8192 || m3.GetTokens() != MaxChunkTokens {
		t.Errorf("bge-m3 tokens = %d of %d", m3.GetTokens(), m3.MaxTokens)
	}
}

func TestHandlingApplyAddsTheRolePrefixInFrontOfEveryText(t *testing.T) {
	h := InputFormat{QueryPrefix: "query: ", DocumentPrefix: "passage: "}
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
	if got := (InputFormat{}).Apply(RoleQuery, texts); !slices.Equal(got, texts) || &got[0] != &texts[0] {
		t.Errorf("no prefix changed or reallocated the texts: %q", got)
	}
}

func TestEveryCuratedModelWithAQueryPrefixAppliesItToQueries(t *testing.T) {
	for _, p := range getShippedProfiles(t).GetCurated() {
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
	base := InputFormat{QueryPrefix: "query: ", DocumentPrefix: "passage: ", File: "model.onnx", Normalize: true, Tokenizer: "1"}
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
		"query prefix":     func() string { h := base; h.QueryPrefix = "q: "; return h.Hash(512) }(),
		"document prefix":  func() string { h := base; h.DocumentPrefix = "d: "; return h.Hash(512) }(),
		"file":             func() string { h := base; h.File = "model_q4.onnx"; return h.Hash(512) }(),
		"normalize":        func() string { h := base; h.Normalize = false; return h.Hash(512) }(),
		"max tokens":       base.Hash(256),
		"tokenizer":        func() string { h := base; h.Tokenizer = "2"; return h.Hash(512) }(),
		"absent tokenizer": func() string { h := base; h.Tokenizer = ""; return h.Hash(512) }(),
	}
	for what, got := range changed {
		if got == want {
			t.Errorf("a change of the %s leaves the hash %s", what, want)
		}
	}
}

func TestHandlingIsZeroOnlyWhenItChangesNothing(t *testing.T) {
	if !(InputFormat{}).IsZero() {
		t.Error("an empty handling is not zero")
	}
	for _, h := range []InputFormat{{QueryPrefix: "q"}, {DocumentPrefix: "d"}, {File: "f"}, {Normalize: true}, {ExtraInputs: []string{InputPositionIDs}}, {Tokenizer: "1"}} {
		if h.IsZero() {
			t.Errorf("%+v is zero", h)
		}
	}
}

// The hash of a handling with no tokenizer was recorded before the tokenizer joined the handling: a
// connection and the mock model keep their recipe, so nothing they embedded is embedded again.
func TestAHandlingWithoutATokenizerHashesAsBefore(t *testing.T) {
	h := InputFormat{QueryPrefix: "query: ", DocumentPrefix: "passage: ", File: "model.onnx", Normalize: true}
	if got, want := h.Hash(512), "c69543d1"; got != want {
		t.Errorf("hash = %s, want %s as before the tokenizer joined the handling", got, want)
	}
}

func TestOptionTextsNameTierSizeAndWidth(t *testing.T) {
	p, err := getShippedProfiles(t).GetLocal(DefaultModel)
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
	if table, err := GetShippedProfiles(); table == nil || err == nil || errors.Unwrap(err) == nil {
		t.Errorf("GetShippedProfiles on broken JSON = (%v, %v), want non-nil table and wrapped error", table, err)
	}
	embeddedProfiles = []byte(`[{"id":"x","dims":0}]`)
	if table, err := GetShippedProfiles(); table == nil || err == nil || errors.Unwrap(err) == nil {
		t.Errorf("GetShippedProfiles on invalid profile = (%v, %v), want non-nil table and wrapped error", table, err)
	}
	embeddedProfiles = origEmbedded

	table := getShippedProfiles(t)
	withExtra := table.WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "extra.json", Data: []byte(`{"id":"Xenova/bge-m3","extraInputs":["position_ids"]}`)}},
		func(string, error) {},
	)
	m3, err := withExtra.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	m3.Aliases[0] = "changed"
	m3.ExtraInputs[0] = "changed"
	m3.Dependencies[TokenizerFile] = "changed"

	m3Again, err := withExtra.GetLocal("Xenova/bge-m3")
	if err != nil {
		t.Fatal(err)
	}
	if m3Again.Aliases[0] == "changed" || m3Again.ExtraInputs[0] == "changed" || m3Again.Dependencies[TokenizerFile] == "changed" {
		t.Errorf("mutating returned Profile mutated table state: %+v", m3Again)
	}
}

func getTokenizerKind(t *testing.T, p Profile) string {
	t.Helper()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var fields struct {
		Tokenizer string `json:"tokenizer"`
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	return fields.Tokenizer
}

func TestShippedProfilesNameTheKindOfTheirTokenizer(t *testing.T) {
	table := getShippedProfiles(t)
	want := map[string]string{
		"Xenova/multilingual-e5-small":            "unigram",
		"Xenova/bge-m3":                           "unigram",
		"onnx-community/harrier-oss-v1-270m-ONNX": "bpe",
	}
	for id, kind := range want {
		p, err := table.GetLocal(id)
		if err != nil {
			t.Fatal(err)
		}
		if got := getTokenizerKind(t, p); got != kind {
			t.Errorf("%s: tokenizer %q, want %q", id, got, kind)
		}
	}
}

func TestAUserProfileThatAddsAModelWithoutATokenizerIsReportedAndSkipped(t *testing.T) {
	var reported error
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "mine.json", Data: []byte(`{"id":"owner/mine","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx"}`)}},
		func(_ string, err error) { reported = err },
	)
	if reported == nil || !strings.Contains(reported.Error(), "tokenizer") {
		t.Errorf("reported %v, want an error that names the tokenizer", reported)
	}
	if _, err := table.GetLocal("owner/mine"); err == nil {
		t.Error("a model without a tokenizer is in the table")
	}
}

func TestAUserProfileWithAnUnknownTokenizerIsReportedAndSkipped(t *testing.T) {
	var reported error
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]UserFile{{Path: "mine.json", Data: []byte(`{"id":"owner/mine","dims":8,"maxTokens":8,"pooling":"mean","onnxFile":"m.onnx","tokenizer":"wordpiece"}`)}},
		func(_ string, err error) { reported = err },
	)
	if reported == nil || !strings.Contains(reported.Error(), "wordpiece") {
		t.Errorf("reported %v, want an error that names the tokenizer wordpiece", reported)
	}
	if _, err := table.GetLocal("owner/mine"); err == nil {
		t.Error("a model with an unknown tokenizer is in the table")
	}
}

func TestAUserProfileMergedOverAShippedOneKeepsItsTokenizer(t *testing.T) {
	var reported []string
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]UserFile{
			{Path: "e5.json", Data: []byte(`{"id":"Xenova/multilingual-e5-small","documentPrefix":"doc: "}`)},
			{Path: "harrier.json", Data: []byte(`{"id":"onnx-community/harrier-oss-v1-270m-ONNX","queryPrefix":"Q: "}`)},
		},
		func(file string, _ error) { reported = append(reported, file) },
	)
	if len(reported) != 0 {
		t.Fatalf("reported %v", reported)
	}
	for id, kind := range map[string]string{DefaultModel: "unigram", "onnx-community/harrier-oss-v1-270m-ONNX": "bpe"} {
		p, err := table.GetLocal(id)
		if err != nil {
			t.Fatal(err)
		}
		if got := getTokenizerKind(t, p); got != kind {
			t.Errorf("%s: tokenizer %q after a merge, want %q", id, got, kind)
		}
	}
}
