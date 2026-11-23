package searchfunction_test

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction/searchfunctiontest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
)

const token = "shelf-token"

func newShelfClient(t *testing.T, calls chan<- searchfunction.Call) *searchfunctiontest.Client {
	t.Helper()
	shelves := map[string][]string{"east": {"a.md", "b.md", "c.md"}}
	registry := searchfunction.New()
	err := registry.Add("shelved", searchfunction.Function{
		Match: func(_ context.Context, call searchfunction.Call, _ []string) (searchfunction.MatchResult, error) {
			calls <- call
			shelf, _ := call.Arguments.GetString("shelf")
			if shelf == "" {
				return searchfunction.MatchResult{}, fmt.Errorf("%w: shelved needs a shelf", searchfunction.ErrInvalidArguments)
			}
			return searchfunction.MatchResult{Paths: append(slices.Clone(shelves[shelf]), "unasked.md"), Warning: "late"}, nil
		},
		Find: func(_ context.Context, call searchfunction.Call, _ int) (searchfunction.Found, error) {
			calls <- call
			shelf, _ := call.Arguments.GetString("shelf")
			return searchfunction.Found{Paths: shelves[shelf], EstimatedCount: int64(len(shelves[shelf]))}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Add("shelved", searchfunction.Function{}); !errors.Is(err, searchfunction.ErrDuplicate) {
		t.Fatalf("a second shelved returned %v, want ErrDuplicate", err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, func(s *server.Server) error {
		registry.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return searchfunctiontest.NewClient(httpServer.Client(), httpServer.URL, token)
}

func TestAFunctionReadsItsArgumentsByNameAndMatchDropsUnaskedPaths(t *testing.T) {
	calls := make(chan searchfunction.Call, 4)
	client := newShelfClient(t, calls)

	paths, warning, err := client.MatchDocuments(t.Context(), searchfunctiontest.Call{
		Function: "shelved", VaultID: "v1", IndexGeneration: "g2", Arguments: map[string]any{"shelf": "east", "depth": int64(3), "isDeep": true},
	}, []string{"a.md", "c.md", "z.md"})
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a.md", "c.md"}) || warning != "late" {
		t.Errorf("matched %v with %q, want a.md and c.md with the warning", paths, warning)
	}
	call := <-calls
	depth, isInt := call.Arguments.GetInt("depth")
	isDeep, isBool := call.Arguments.GetBool("isDeep")
	if _, isString := call.Arguments.GetString("depth"); isString || !isInt || depth != 3 || !isBool || !isDeep ||
		call.VaultID != "v1" || call.IndexGeneration != "g2" {
		t.Errorf("the function received %+v", call)
	}
}

func TestFindAnswersAtMostTheLimitWithTheCount(t *testing.T) {
	client := newShelfClient(t, make(chan searchfunction.Call, 4))

	paths, count, _, err := client.FindDocuments(t.Context(), searchfunctiontest.Call{Function: "shelved", Arguments: map[string]any{"shelf": "east"}}, 2)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a.md", "b.md"}) || count != 3 {
		t.Errorf("found %v of %d, want two of three", paths, count)
	}
}

func TestAnUnknownFunctionIsNotFoundAndInvalidArgumentsAreInvalidArgument(t *testing.T) {
	client := newShelfClient(t, make(chan searchfunction.Call, 4))

	_, _, err := client.MatchDocuments(t.Context(), searchfunctiontest.Call{Function: "boxed"}, []string{"a.md"})
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("an unknown function returned %v, want NotFound", err)
	}
	_, _, err = client.MatchDocuments(t.Context(), searchfunctiontest.Call{Function: "shelved", Arguments: map[string]any{"shelf": ""}}, []string{"a.md"})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("an empty shelf returned %v, want InvalidArgument", err)
	}
}
