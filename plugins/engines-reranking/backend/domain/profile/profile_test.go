package profile_test

import (
	"errors"
	"maps"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

const (
	gteID = "onnx-community/gte-multilingual-reranker-base"
	bgeID = "onnx-community/bge-reranker-v2-m3-ONNX"
)

func getShippedProfiles(t *testing.T) *profile.Profiles {
	t.Helper()
	table, err := profile.GetShippedProfiles()
	if err != nil {
		t.Fatalf("GetShippedProfiles: %v", err)
	}
	return table
}

func getLocal(t *testing.T, table *profile.Profiles, id string) profile.Profile {
	t.Helper()
	p, err := table.GetLocal(id)
	if err != nil {
		t.Fatalf("GetLocal(%q): %v", id, err)
	}
	return p
}

func ignoreReports(string, error) {}

func TestTheShippedTableListsBGEFirstAndGTESecond(t *testing.T) {
	var ids []string
	for _, p := range getShippedProfiles(t).GetCurated() {
		ids = append(ids, p.ID)
	}

	if !slices.Equal(ids, []string{bgeID, gteID}) {
		t.Errorf("curated profiles = %v, want bge then gte", ids)
	}
	if profile.DefaultModel != bgeID {
		t.Errorf("DefaultModel = %q, want %q", profile.DefaultModel, bgeID)
	}
}

func TestShippedProfilesNameTheFilesTheManifestDeclares(t *testing.T) {
	want := map[string]map[string]string{
		gteID: {"onnx/model_quantized.onnx": "gte-reranker-graph", "tokenizer.json": "gte-reranker-tokenizer"},
		bgeID: {"onnx/model_quantized.onnx": "bge-reranker-v2-m3-graph", "tokenizer.json": "bge-reranker-v2-m3-tokenizer"},
	}
	table := getShippedProfiles(t)

	for id, files := range want {
		p := getLocal(t, table, id)
		if err := p.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
		if p.OnnxFile != "onnx/model_quantized.onnx" || p.Revision != "" || p.SHA256 != nil || !maps.Equal(p.Dependencies, files) {
			t.Errorf("%s fetches %q at %q with %v and %v, want %v", id, p.OnnxFile, p.Revision, p.SHA256, p.Dependencies, files)
		}
		if p.MaxTokens != 512 {
			t.Errorf("%s reads %d tokens, want 512", id, p.MaxTokens)
		}
	}
}

func TestOptionTitlesNameTheTierAndTheSize(t *testing.T) {
	var titles []string
	for _, p := range getShippedProfiles(t).GetCurated() {
		titles = append(titles, p.GetOptionTitle())
	}

	want := []string{"BGE reranker v2 M3 (default, 571 MB)", "GTE multilingual reranker (smaller, 341 MB)"}
	if !slices.Equal(titles, want) {
		t.Errorf("titles = %q, want %q", titles, want)
	}
}

func TestGetLocalReturnsTheDefaultForAnEmptyModel(t *testing.T) {
	table := getShippedProfiles(t)

	for _, name := range []string{"", "  "} {
		if p := getLocal(t, table, name); p.ID != profile.DefaultModel {
			t.Errorf("GetLocal(%q) = %q, want the default %q", name, p.ID, profile.DefaultModel)
		}
	}
}

func TestGetLocalAcceptsAnAliasAndAnyCase(t *testing.T) {
	table := getShippedProfiles(t)
	const paddedRepo = " ONNX-COMMUNITY/bge-reranker-v2-m3-onnx "

	for name, id := range map[string]string{
		"gte-multilingual-reranker-base": gteID,
		"GTE-Multilingual-Reranker-Base": gteID,
		"bge-reranker-v2-m3":             bgeID,
		paddedRepo:                       bgeID,
	} {
		if p := getLocal(t, table, name); p.ID != id {
			t.Errorf("GetLocal(%q) = %q, want %q", name, p.ID, id)
		}
	}
}

func TestGetLocalOfAnUnknownModelNamesTheFolderAndTheKnownModels(t *testing.T) {
	table := getShippedProfiles(t).WithUserProfiles("/data/profiles", nil, ignoreReports)

	_, err := table.GetLocal("owner/unknown-reranker")

	if !errors.Is(err, profile.ErrNoProfile) {
		t.Fatalf("GetLocal error = %v, want ErrNoProfile", err)
	}
	for _, part := range []string{"/data/profiles", gteID, bgeID} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("error %q does not name %s", err, part)
		}
	}
	if profile.ProfilesFolder != "profiles" {
		t.Errorf("ProfilesFolder = %q, want profiles", profile.ProfilesFolder)
	}
}

func TestReadProfilesReadsAnArrayOrOneProfile(t *testing.T) {
	one, err := profile.ReadProfiles([]byte(`{"id":"owner/lantern","title":"Lantern","maxTokens":256,"onnxFile":"onnx/model.onnx"}`))
	if err != nil || len(one) != 1 || one[0].ID != "owner/lantern" || one[0].MaxTokens != 256 {
		t.Errorf("one profile = %+v, %v", one, err)
	}

	two, err := profile.ReadProfiles([]byte(`[{"id":"owner/lantern","maxTokens":256,"onnxFile":"m.onnx"},{"id":"owner/harbor","maxTokens":128,"onnxFile":"m.onnx"}]`))
	if err != nil || len(two) != 2 || two[1].ID != "owner/harbor" {
		t.Errorf("two profiles = %+v, %v", two, err)
	}
}

func TestReadProfilesRejectsAnUnknownField(t *testing.T) {
	for _, field := range []string{`"pooling":"cls"`, `"score":"softmax"`, `"positiveLabel":1`, `"instruction":"Find the notes"`} {
		_, err := profile.ReadProfiles([]byte(`{"id":"owner/lantern","maxTokens":256,"onnxFile":"m.onnx",` + field + `}`))
		if err == nil {
			t.Errorf("a profile with %s was read", field)
		}
	}
}

func TestValidateRefusesWhatCannotRun(t *testing.T) {
	good := profile.Profile{ID: "owner/lantern", Title: "Lantern", MaxTokens: 9, OnnxFile: "onnx/model.onnx"}
	if err := good.Validate(); err != nil {
		t.Fatalf("a profile that can run: %v", err)
	}

	for name, broken := range map[string]profile.Profile{
		"blank id":        {ID: "  ", MaxTokens: 512, OnnxFile: "onnx/model.onnx"},
		"8 tokens":        {ID: "owner/lantern", MaxTokens: 8, OnnxFile: "onnx/model.onnx"},
		"no tokens":       {ID: "owner/lantern", OnnxFile: "onnx/model.onnx"},
		"no model file":   {ID: "owner/lantern", MaxTokens: 512},
		"negative tokens": {ID: "owner/lantern", MaxTokens: -1, OnnxFile: "onnx/model.onnx"},
	} {
		if err := broken.Validate(); err == nil {
			t.Errorf("%s: Validate accepted %+v", name, broken)
		}
	}
}

func TestUserProfileMergesOverTheShippedOneByAlias(t *testing.T) {
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]profile.UserFile{{Path: "bge.json", Data: []byte(`{"id":"bge-reranker-v2-m3","maxTokens":256}`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)

	p := getLocal(t, table, bgeID)

	if p.MaxTokens != 256 {
		t.Errorf("maxTokens = %d, want the 256 of the file", p.MaxTokens)
	}
	if p.Title != "BGE reranker v2 M3" || p.Tier != "default" || len(p.Dependencies) != 2 {
		t.Errorf("a field the file leaves out was lost: %+v", p)
	}
	if n := len(table.GetCurated()); n != 2 {
		t.Errorf("an alias added a model: %d curated", n)
	}
	if shipped := getLocal(t, getShippedProfiles(t), bgeID); shipped.MaxTokens != 512 {
		t.Error("the file changed the shipped table")
	}
}

func TestUserProfileAddsAModel(t *testing.T) {
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]profile.UserFile{{Path: "lantern.json", Data: []byte(`{"id":"owner/lantern-reranker","title":"Lantern","maxTokens":384,"onnxFile":"onnx/model.onnx"}`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)

	p := getLocal(t, table, "owner/lantern-reranker")

	if p.MaxTokens != 384 || p.OnnxFile != "onnx/model.onnx" || p.Tier != "" {
		t.Errorf("profile = %+v", p)
	}
	if n := len(table.GetCurated()); n != 2 {
		t.Errorf("a profile with no tier is listed: %d curated", n)
	}
}

func TestUserProfileThatChangesTheFileDropsThePins(t *testing.T) {
	table := getShippedProfiles(t).WithUserProfiles(
		"/data/profiles",
		[]profile.UserFile{{Path: "gte.json", Data: []byte(`{"id":"` + gteID + `","onnxFile":"onnx/model_fp16.onnx"}`)}},
		func(file string, err error) { t.Errorf("reported %s: %v", file, err) },
	)

	p := getLocal(t, table, gteID)

	if p.OnnxFile != "onnx/model_fp16.onnx" || p.Revision != "" || p.SHA256 != nil || p.Dependencies != nil {
		t.Errorf("the pins of the shipped file stay: %+v", p)
	}
}

func TestMalformedUserFilesAreReportedAndSkipped(t *testing.T) {
	files := []profile.UserFile{
		{Path: "a-broken.json", Data: []byte(`{"id": "owner/x", `)},
		{Path: "b-unknown-field.json", Data: []byte(`{"id":"owner/y","maxTokens":256,"onnxFile":"m.onnx","pooling":"cls"}`)},
		{Path: "c-score.json", Data: []byte(`{"id":"owner/z","maxTokens":256,"onnxFile":"m.onnx","score":"softmax"}`)},
		{Path: "d-half.json", Data: []byte(`[{"id":"owner/ok","maxTokens":256,"onnxFile":"m.onnx"},{"id":"owner/bad","maxTokens":4,"onnxFile":"m.onnx"}]`)},
		{Path: "e-good.json", Data: []byte(`{"id":"owner/good","maxTokens":256,"onnxFile":"m.onnx"}`)},
	}
	reported := map[string]error{}

	table := getShippedProfiles(t).WithUserProfiles("/data/profiles", files, func(file string, err error) { reported[file] = err })

	for _, name := range []string{"a-broken.json", "b-unknown-field.json", "c-score.json", "d-half.json"} {
		if reported[name] == nil {
			t.Errorf("%s was not reported", name)
		}
	}
	if len(reported) != 4 {
		t.Errorf("reported %v, want the four broken files", reported)
	}
	for _, id := range []string{"owner/x", "owner/y", "owner/z", "owner/ok", "owner/bad"} {
		if _, err := table.GetLocal(id); err == nil {
			t.Errorf("%s from a skipped file is listed", id)
		}
	}
	getLocal(t, table, "owner/good")
	getLocal(t, table, profile.DefaultModel)
}

func TestGetScoreIsTheSigmoidOfTheLogit(t *testing.T) {
	for logit, want := range map[float32]float64{0: 0.5, 2: 1 / (1 + math.Exp(-2)), -3: 1 / (1 + math.Exp(3)), 40: 1, -40: 0} {
		if got := profile.GetScore(logit); math.Abs(float64(got)-want) > 1e-6 {
			t.Errorf("GetScore(%v) = %v, want %v", logit, got, want)
		}
	}
}
