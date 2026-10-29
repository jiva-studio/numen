package api_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

// messageTypes maps each method to a constructor of its request and of its response.
var messageTypes = map[string]struct{ request, response func() any }{
	"ListCardModels":       {func() any { return new(api.ListCardModelsRequest) }, func() any { return new(api.ListCardModelsResponse) }},
	"CreateCardModel":      {func() any { return new(api.CreateCardModelRequest) }, func() any { return new(api.CreateCardModelResponse) }},
	"ReadCardModel":        {func() any { return new(api.ReadCardModelRequest) }, func() any { return new(api.ReadCardModelResponse) }},
	"WriteCardModel":       {func() any { return new(api.WriteCardModelRequest) }, func() any { return new(api.WriteCardModelResponse) }},
	"RenameCardModelField": {func() any { return new(api.RenameCardModelFieldRequest) }, func() any { return new(api.RenameCardModelFieldResponse) }},
	"CreateDeck":           {func() any { return new(api.CreateDeckRequest) }, func() any { return new(api.CreateDeckResponse) }},
	"ReadDeck":             {func() any { return new(api.ReadDeckRequest) }, func() any { return new(api.ReadDeckResponse) }},
	"WriteDeck":            {func() any { return new(api.WriteDeckRequest) }, func() any { return new(api.WriteDeckResponse) }},
	"GetDeckPreset":        {func() any { return new(api.GetDeckPresetRequest) }, func() any { return new(api.GetDeckPresetResponse) }},
	"ListPresets":          {func() any { return new(api.ListPresetsRequest) }, func() any { return new(api.ListPresetsResponse) }},
	"CreatePreset":         {func() any { return new(api.CreatePresetRequest) }, func() any { return new(api.CreatePresetResponse) }},
	"ScheduleDeck":         {func() any { return new(api.ScheduleDeckRequest) }, func() any { return new(api.ScheduleDeckResponse) }},
	"ReadPreset":           {func() any { return new(api.ReadPresetRequest) }, func() any { return new(api.ReadPresetResponse) }},
	"WritePreset":          {func() any { return new(api.WritePresetRequest) }, func() any { return new(api.WritePresetResponse) }},
	"ComputeCurve":         {func() any { return new(api.ComputeCurveRequest) }, func() any { return new(api.ComputeCurveResponse) }},
	"WatchCardsDue":        {func() any { return new(api.WatchCardsDueRequest) }, func() any { return new(api.WatchCardsDueResponse) }},
	"StartSession":         {func() any { return new(api.StartSessionRequest) }, func() any { return new(api.StartSessionResponse) }},
	"AnswerCard":           {func() any { return new(api.AnswerCardRequest) }, func() any { return new(api.AnswerCardResponse) }},
	"TakeBackAnswer":       {func() any { return new(api.TakeBackAnswerRequest) }, func() any { return new(api.TakeBackAnswerResponse) }},
	"WatchReloads":         {func() any { return new(api.WatchReloadsRequest) }, func() any { return new(api.WatchReloadsResponse) }},
	"ListReviewDays":       {func() any { return new(api.ListReviewDaysRequest) }, func() any { return new(api.ListReviewDaysResponse) }},
	"GetDeckNeighbourhood": {func() any { return new(api.GetDeckNeighbourhoodRequest) }, func() any { return new(api.GetDeckNeighbourhoodResponse) }},
	"GetVaultDeckPreset":   {func() any { return new(api.GetVaultDeckPresetRequest) }, func() any { return new(api.GetVaultDeckPresetResponse) }},
}

// normalize returns the canonical JSON of raw, so that two documents with the same content and a
// different layout compare equal.
func normalize(t *testing.T, raw []byte) string {
	t.Helper()
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestEveryMethodHasFixtures(t *testing.T) {
	for method := range messageTypes {
		for _, kind := range []string{"request", "response"} {
			path := filepath.Join("..", "..", "testdata", "api", method+"."+kind+".json")
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s has no %s fixture: %v", method, kind, err)
			}
		}
	}
}

func TestFixturesRoundTripThroughTheStructs(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "testdata", "api", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no fixtures")
	}
	for _, path := range paths {
		name := filepath.Base(path)
		parts := strings.Split(strings.TrimSuffix(name, ".json"), ".")
		method, kind := parts[0], parts[len(parts)-1]
		types, isDeclared := messageTypes[method]
		if !isDeclared {
			t.Errorf("%s names no method", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			message := types.request()
			if kind == "response" {
				message = types.response()
			}
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(message); err != nil {
				t.Fatalf("decode: %v", err)
			}
			api.FillCollections(message)
			encoded, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := normalize(t, encoded), normalize(t, raw); got != want {
				t.Errorf("re-encoded\n got %s\nwant %s", got, want)
			}
		})
	}
}
