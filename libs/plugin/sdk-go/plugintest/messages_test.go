package plugintest

import (
	"os"
	"path/filepath"
	"testing"
)

type pingRequest struct {
	Name string `json:"name"`
}

type pingResponse struct {
	Count int `json:"count"`
}

func createFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"Ping.request.json":  `{"name": "a"}`,
		"Ping.response.json": `{"count": 2}`,
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func createPingContracts() map[string]MessageContract {
	return map[string]MessageContract{
		"Ping": {
			Request:  func() any { return &pingRequest{} },
			Response: func() any { return &pingResponse{} },
		},
	}
}

func TestCheckMessageFixturesAcceptsMatchingFixtures(t *testing.T) {
	CheckMessageFixtures(t, createFixtureDir(t), createPingContracts())
}

func TestCheckFixturePairsAcceptsOnePairPerMethod(t *testing.T) {
	CheckFixturePairs(t, createFixtureDir(t), createPingContracts())
}

func TestCheckSameJSONIgnoresKeyOrderAndSpacing(t *testing.T) {
	checkSameJSON(t, []byte(`{"a": 1, "b": 2}`), []byte(`{"b":2,"a":1}`))
}
