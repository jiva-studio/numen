package onnx

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
)

// RepositoryModel describes an ONNX graph and its tokenizer in a Hugging Face repository.
type RepositoryModel struct {
	Endpoint       string
	ID             string
	Revision       string
	Graph          string
	Sums           map[string]string
	Files          []string
	SelectExternal func(string) bool
}

// ModelFiles holds the graph and tokenizer locations used to open a model.
type ModelFiles struct {
	Model     string
	Tokenizer string
}

// FetchFile resolves a remote file to a local path. An empty digest requests the file without an author pin.
type FetchFile func(context.Context, string, string) (string, error)

// PrepareModel fetches the selected graph, its external data and the tokenizer. External data
// files are placed next to the graph under the relative names encoded in the model.
func PrepareModel(ctx context.Context, model RepositoryModel, cache string, fetch FetchFile) (ModelFiles, error) {
	if !isRepositoryPath(model.ID) || !isRepositoryPath(model.Graph) {
		return ModelFiles{}, fmt.Errorf("invalid model repository %q or graph %q", model.ID, model.Graph)
	}
	if model.Endpoint == "" {
		model.Endpoint = "https://huggingface.co"
	}
	revision := model.Revision
	if revision == "" {
		revision = "main"
	}
	files, revision, err := model.listFiles(ctx, revision, fetch)
	if err != nil {
		return ModelFiles{}, err
	}
	locals := make(map[string]string, len(files))
	for _, name := range slices.Sorted(maps.Keys(files)) {
		address, err := url.JoinPath(model.Endpoint, model.ID, "resolve", url.PathEscape(revision), name)
		if err != nil {
			return ModelFiles{}, err
		}
		file, err := fetch(ctx, address, files[name])
		if err != nil {
			return ModelFiles{}, fmt.Errorf("fetch %s of %s: %w", name, model.ID, err)
		}
		locals[name] = file
	}
	tokens := locals["tokenizer.json"]
	delete(locals, "tokenizer.json")
	graph, err := PlaceModelFiles(cache, model.Graph, locals)
	return ModelFiles{Model: graph, Tokenizer: tokens}, err
}

func (model RepositoryModel) listFiles(ctx context.Context, revision string, fetch FetchFile) (map[string]string, string, error) {
	if len(model.Files) > 0 {
		files := make(map[string]string)
		for _, name := range model.Files {
			if model.isMember(name, model.Files) {
				files[name] = model.Sums[name]
			}
		}
		return files, revision, nil
	}
	address, err := url.JoinPath(model.Endpoint, "api/models", model.ID, "revision", url.PathEscape(revision))
	if err != nil {
		return nil, "", err
	}
	index, err := fetch(ctx, address+"?blobs=true", "")
	if err != nil {
		return nil, "", err
	}
	return model.readFiles(index)
}

func (model RepositoryModel) isMember(name string, names []string) bool {
	if !isRepositoryPath(name) || strings.HasPrefix(name, model.Graph+"/") {
		return false
	}
	if name == "tokenizer.json" || name == model.Graph {
		return true
	}
	folder := path.Dir(model.Graph)
	if folder != "." && !strings.HasPrefix(name, folder+"/") {
		return false
	}
	if _, pinned := model.Sums[name]; !pinned && model.SelectExternal != nil && !model.SelectExternal(name) {
		return false
	}
	return !model.isInOtherGraph(name, names)
}

func isRepositoryPath(name string) bool {
	return fs.ValidPath(name) && name != "." && !strings.ContainsAny(name, "\\?#")
}

func (model RepositoryModel) isInOtherGraph(name string, names []string) bool {
	for _, other := range names {
		if other != model.Graph && strings.HasSuffix(other, ".onnx") && strings.HasPrefix(name, other) {
			return true
		}
	}
	return false
}

func (model RepositoryModel) readFiles(index string) (map[string]string, string, error) {
	raw, err := os.ReadFile(index)
	if err != nil {
		return nil, "", err
	}
	var info struct {
		Revision string `json:"sha"`
		Files    []struct {
			Name string `json:"rfilename"`
			LFS  struct {
				SHA256 string `json:"sha256"`
			} `json:"lfs"`
		} `json:"siblings"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, "", err
	}
	if info.Revision == "" {
		return nil, "", fmt.Errorf("%s returned no revision", model.ID)
	}
	names := make([]string, 0, len(info.Files))
	for _, entry := range info.Files {
		names = append(names, entry.Name)
	}
	files := make(map[string]string)
	for _, entry := range info.Files {
		if model.isMember(entry.Name, names) {
			sum := model.Sums[entry.Name]
			if sum == "" {
				sum = entry.LFS.SHA256
			}
			files[entry.Name] = sum
		}
	}
	for _, name := range []string{model.Graph, "tokenizer.json"} {
		if _, found := files[name]; !found {
			return nil, "", fmt.Errorf("%s publishes no %s", model.ID, name)
		}
	}
	return files, info.Revision, nil
}
