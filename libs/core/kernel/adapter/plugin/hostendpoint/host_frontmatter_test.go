package hostendpoint

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/frontmatter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
)

// storedFrontmatter is the frontmatter of the notes of a vault as an index holds it.
type storedFrontmatter map[string]map[string]any

func (s storedFrontmatter) ListFrontmatter(_ context.Context, _ domain.VaultID, q frontmatter.Query) ([]frontmatter.Note, error) {
	paths := q.Paths
	if len(paths) == 0 {
		paths = slices.Collect(maps.Keys(s))
	}
	var out []frontmatter.Note
	for _, path := range slices.Compact(slices.Sorted(slices.Values(paths))) {
		values, ok := s[path]
		if !ok || path <= q.After || len(out) == q.Limit {
			continue
		}
		picked := map[string]any{}
		for key, value := range values {
			if len(q.Keys) == 0 || slices.Contains(q.Keys, key) {
				picked[key] = value
			}
		}
		out = append(out, frontmatter.Note{Path: path, Values: picked})
	}
	return out, nil
}

// readingFrontmatter is a vault whose frontmatter the use case reads, over the notes it stores.
type readingFrontmatter struct {
	read query.ReadFrontmatter
}

func (r readingFrontmatter) ReadFrontmatter(ctx context.Context, q frontmatter.Query) (frontmatter.Page, error) {
	return r.read.Execute(ctx, domain.Vault{ID: "v1"}, q)
}

func newFrontmatterClient(t *testing.T, token string, stored storedFrontmatter, limits query.FrontmatterLimits) testsupport.HostVaultClient {
	t.Helper()
	hosted := newCallers()
	hosted.vault = vaultOf(readingFrontmatter{read: query.NewReadFrontmatter(stored, &memoryFeed{}, limits)})
	return newVaultClient(t, hosted, token)
}

func readPages(t *testing.T, client testsupport.HostVaultClient, request *v1.ReadFrontmatterRequest) (paths []string, values map[string]map[string]any) {
	t.Helper()
	values = map[string]map[string]any{}
	for range 10 {
		res, err := client.ReadFrontmatter(t.Context(), connect.NewRequest(request))
		if err != nil {
			t.Fatal(err)
		}
		for _, note := range res.Msg.GetNotes() {
			var one map[string]any
			if err := json.Unmarshal(note.GetValues(), &one); err != nil {
				t.Fatal(err)
			}
			paths, values[note.GetPath()] = append(paths, note.GetPath()), one
		}
		if res.Msg.GetNextPageToken() == "" {
			return paths, values
		}
		request.PageToken = res.Msg.GetNextPageToken()
	}
	t.Fatal("the pages never ended")
	return nil, nil
}

func TestReadFrontmatterPagesThroughEveryNoteInPathOrder(t *testing.T) {
	stored := storedFrontmatter{
		"c.md": {"tags": []any{"river"}}, "a.md": {"tags": "lake", "status": "open"}, "b.md": {},
	}
	client := newFrontmatterClient(t, "token-a", stored, query.FrontmatterLimits{PageSize: 2, MaxPaths: 10, MaxKeys: 2})

	paths, values := readPages(t, client, &v1.ReadFrontmatterRequest{Keys: []string{"tags"}})

	if !slices.Equal(paths, []string{"a.md", "b.md", "c.md"}) {
		t.Errorf("read %v, want every note in path order", paths)
	}
	if values["a.md"]["tags"] != "lake" || values["a.md"]["status"] != nil || len(values["b.md"]) != 0 {
		t.Errorf("read the values %v, want only tags", values)
	}
}

func TestReadFrontmatterOverItsLimitsIsAnInvalidArgument(t *testing.T) {
	client := newFrontmatterClient(t, "token-a", storedFrontmatter{}, query.FrontmatterLimits{PageSize: 2, MaxPaths: 2, MaxKeys: 1})

	for name, request := range map[string]*v1.ReadFrontmatterRequest{
		"too many paths": {Paths: []string{"a.md", "b.md", "c.md"}},
		"too many keys":  {Keys: []string{"tags", "status"}},
		"too large page": {PageSize: 3},
	} {
		_, err := client.ReadFrontmatter(t.Context(), connect.NewRequest(request))
		if got := readCode(t, err); got != connect.CodeInvalidArgument {
			t.Errorf("%s answered %v, want an invalid argument", name, got)
		}
	}
}

func TestReadFrontmatterIsRefusedWithoutVaultRead(t *testing.T) {
	client := newFrontmatterClient(t, "token-b", storedFrontmatter{"a.md": {"tags": "lake"}}, query.GetFrontmatterLimits())

	_, err := client.ReadFrontmatter(t.Context(), connect.NewRequest(&v1.ReadFrontmatterRequest{}))

	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("a read without vault:read is %v", got)
	}
}
