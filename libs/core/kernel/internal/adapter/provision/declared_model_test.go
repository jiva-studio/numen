package provision

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugindependency"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const (
	speechPlugin  = "md.example.speech"
	searchPlugin  = "md.example.search"
	speechEncoder = "speech-encoder"
	encoderTitle  = "Speech encoder"
	declaredWait  = 5 * time.Second
)

func getEncoderBytes() []byte {
	return []byte("weights of an example speech encoder")
}

type declaredManifests map[string]manifest.Manifest

func (d declaredManifests) FindModelDependency(pluginID, id string) (manifest.DependencyRequirement, bool) {
	m, ok := d[pluginID]
	if !ok {
		return manifest.DependencyRequirement{}, false
	}
	return m.FindModelDependency(id)
}

type modelServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string
}

func newModelServer(t *testing.T, files map[string][]byte) *modelServer {
	t.Helper()
	s := &modelServer{}
	s.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.URL.Path)
		s.mu.Unlock()
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if _, err := w.Write(body); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *modelServer) getRequests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

func declareModel(id, title string, data []byte, addresses ...string) string {
	return fmt.Sprintf(`{"id": %q, "kind": "model", "version": "r1", "sourceUrls": [%s], "checksumSha256": %q, "sizeBytes": %d, "metadata": {"title": %q}}`,
		id, `"`+strings.Join(addresses, `", "`)+`"`, sumOf(data), len(data), title)
}

func readDeclaredManifest(t *testing.T, pluginID string, dependencies ...string) manifest.Manifest {
	t.Helper()
	m, err := manifestjson.Read(strings.NewReader(fmt.Sprintf(
		`{"id": %q, "name": "Example", "version": "1.0.0", "backend": {"entrypoint": "bin/example"}, "dependencies": [%s]}`,
		pluginID, strings.Join(dependencies, ", "),
	)))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newDeclaredFetch(manifests declaredManifests, registry *Registry) *plugindependency.FetchDeclaredModel {
	return plugindependency.NewFetchDeclaredModel(manifests, func() pluginport.DependencyRegistry { return registry })
}

func newSpeechFetch(t *testing.T, server *modelServer, offline bool, addresses ...string) (*plugindependency.FetchDeclaredModel, *FileProvisioner, *Registry) {
	t.Helper()
	files := newTestProvisioner(t, offline)
	files.cfg.Client = server.Client()
	registry := NewObservedRegistry(nil, files)
	manifests := declaredManifests{speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel(speechEncoder, encoderTitle, getEncoderBytes(), addresses...))}
	return newDeclaredFetch(manifests, registry), files, registry
}

func readModelFile(t *testing.T, path string) []byte {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func listStoredFiles(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			paths = append(paths, path)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestADeclaredModelIsFetchedFromItsFirstSource(t *testing.T) {
	server := newModelServer(t, map[string][]byte{"/first/encoder.onnx": getEncoderBytes(), "/mirror/encoder.onnx": getEncoderBytes()})
	fetch, _, _ := newSpeechFetch(t, server, false, server.URL+"/first/encoder.onnx", server.URL+"/mirror/encoder.onnx")

	path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)

	if err != nil {
		t.Fatal(err)
	}
	if got := readModelFile(t, path); !bytes.Equal(got, getEncoderBytes()) {
		t.Errorf("the file holds %q, want the declared model", got)
	}
	if got := server.getRequests(); !slices.Equal(got, []string{"/first/encoder.onnx"}) {
		t.Errorf("requests = %v, want the first source only", got)
	}
}

func TestAModelTheManifestOfThePluginDoesNotDeclareIsRefused(t *testing.T) {
	graph := []byte("weights of an example search graph")
	server := newModelServer(t, map[string][]byte{"/encoder.onnx": getEncoderBytes(), "/graph.onnx": graph, "/lib.so": []byte("lib")})
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	runtimeEntry := fmt.Sprintf(`{"id": "speech-runtime", "kind": "runtime", "sourceUrl": %q, "checksumSha256": %q}`, server.URL+"/lib.so", sumOf([]byte("lib")))
	manifests := declaredManifests{
		speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel(speechEncoder, encoderTitle, getEncoderBytes(), server.URL+"/encoder.onnx"), runtimeEntry),
		searchPlugin: readDeclaredManifest(t, searchPlugin, declareModel("search-graph", "Search graph", graph, server.URL+"/graph.onnx")),
	}
	fetch := newDeclaredFetch(manifests, NewObservedRegistry(nil, files))

	for _, asked := range []struct{ pluginID, id string }{
		{speechPlugin, "absent-model"},
		{speechPlugin, "speech-runtime"},
		{speechPlugin, "search-graph"},
		{"md.example.unknown", speechEncoder},
		{speechPlugin, ""},
	} {
		path, err := fetch.Execute(t.Context(), asked.pluginID, asked.id)
		if !errors.Is(err, plugindependency.ErrUndeclaredDependency) || path != "" {
			t.Errorf("%s asking for %q = %q, %v, want ErrUndeclaredDependency", asked.pluginID, asked.id, path, err)
		}
	}
	if got := server.getRequests(); len(got) != 0 {
		t.Errorf("refused requests reached the server: %v", got)
	}
}

func TestAModelIsFetchedFromTheMirrorWhenTheFirstSourceFails(t *testing.T) {
	for name, first := range map[string]map[string][]byte{
		"the first source answers 404":        {},
		"the first source serves other bytes": {"/first/encoder.onnx": []byte("tampered bytes of a speech encoder")},
	} {
		t.Run(name, func(t *testing.T) {
			served := map[string][]byte{"/mirror/encoder.onnx": getEncoderBytes()}
			maps.Copy(served, first)
			server := newModelServer(t, served)
			fetch, files, _ := newSpeechFetch(t, server, false, server.URL+"/first/encoder.onnx", server.URL+"/mirror/encoder.onnx")

			path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)

			if err != nil {
				t.Fatal(err)
			}
			if got := readModelFile(t, path); !bytes.Equal(got, getEncoderBytes()) {
				t.Errorf("the file holds %q, want the bytes of the mirror", got)
			}
			if got := server.getRequests(); !slices.Contains(got, "/mirror/encoder.onnx") {
				t.Errorf("requests = %v, want the mirror among them", got)
			}
			for _, stored := range listStoredFiles(t, files.cfg.Dir) {
				if bytes.Contains(readModelFile(t, stored), []byte("tampered")) {
					t.Errorf("the bytes of the failed source are kept in %s", stored)
				}
			}
		})
	}
}

func TestConcurrentRequestsForOneModelMakeOneDownload(t *testing.T) {
	var requests atomic.Int32
	arrived, gate := make(chan struct{}, 8), make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		arrived <- struct{}{}
		select {
		case <-gate:
		case <-r.Context().Done():
			return
		}
		if _, err := w.Write(getEncoderBytes()); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	registry := NewObservedRegistry(nil, files)
	manifests := declaredManifests{speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel(speechEncoder, encoderTitle, getEncoderBytes(), server.URL+"/encoder.onnx"))}
	fetch := newDeclaredFetch(manifests, registry)
	type answer struct {
		path string
		err  error
	}
	answers := make(chan answer, 3)
	for range 3 {
		go func() {
			path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)
			answers <- answer{path, err}
		}()
	}
	select {
	case <-arrived:
	case <-time.After(declaredWait):
		t.Fatal("no download started")
	}
	close(gate)

	var paths []string
	for range 3 {
		got := <-answers
		if got.err != nil {
			t.Fatal(got.err)
		}
		paths = append(paths, got.path)
	}
	if requests.Load() != 1 {
		t.Errorf("three requests at once downloaded %d times, want 1", requests.Load())
	}
	if paths[0] != paths[1] || paths[1] != paths[2] {
		t.Errorf("paths = %v, want one file", paths)
	}
}

func TestTwoPluginsDeclaringOneFileStoreItOnce(t *testing.T) {
	server := newModelServer(t, map[string][]byte{"/speech/graph.onnx": getEncoderBytes(), "/search/graph.onnx": getEncoderBytes()})
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	manifests := declaredManifests{
		speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel("speech-graph", "Speech graph", getEncoderBytes(), server.URL+"/speech/graph.onnx")),
		searchPlugin: readDeclaredManifest(t, searchPlugin, declareModel("graph", "Graph", getEncoderBytes(), server.URL+"/search/graph.onnx")),
	}
	fetch := newDeclaredFetch(manifests, NewObservedRegistry(nil, files))

	first, err := fetch.Execute(t.Context(), speechPlugin, "speech-graph")
	if err != nil {
		t.Fatal(err)
	}
	second, err := fetch.Execute(t.Context(), searchPlugin, "graph")
	if err != nil {
		t.Fatal(err)
	}

	if first != second {
		t.Errorf("one file is stored at %q and at %q", first, second)
	}
	if got := server.getRequests(); len(got) != 1 {
		t.Errorf("requests = %v, want one download", got)
	}
}

func TestAModelFetchedByItsAddressIsNotFetchedAgainByItsID(t *testing.T) {
	server := newModelServer(t, map[string][]byte{"/by-address.onnx": getEncoderBytes(), "/encoder.onnx": getEncoderBytes()})
	fetch, _, registry := newSpeechFetch(t, server, false, server.URL+"/encoder.onnx")
	byAddress := plugindependency.NewFetch(func() pluginport.DependencyRegistry { return registry })
	stored, err := byAddress.Execute(t.Context(), server.URL+"/by-address.onnx", sumOf(getEncoderBytes()))
	if err != nil {
		t.Fatal(err)
	}

	path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)

	if err != nil {
		t.Fatal(err)
	}
	if path != stored {
		t.Errorf("path = %q, want the file already stored at %q", path, stored)
	}
	if got := server.getRequests(); !slices.Equal(got, []string{"/by-address.onnx"}) {
		t.Errorf("requests = %v, want only the fetch by address", got)
	}
}

func TestAnOfflineMachineNamesTheMissingModelAndDownloadsNothing(t *testing.T) {
	server := newModelServer(t, map[string][]byte{"/encoder.onnx": getEncoderBytes()})
	fetch, _, _ := newSpeechFetch(t, server, true, server.URL+"/encoder.onnx")

	path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)

	if !errors.Is(err, ErrOffline) || path != "" {
		t.Fatalf("offline fetch = %q, %v, want ErrOffline", path, err)
	}
	if !strings.Contains(err.Error(), encoderTitle) {
		t.Errorf("the error %q does not name the model %q", err, encoderTitle)
	}
	if got := server.getRequests(); len(got) != 0 {
		t.Errorf("an offline machine made requests %v", got)
	}
}

func TestAnOfflineMachineUsesAModelAlreadyFetched(t *testing.T) {
	server := newModelServer(t, map[string][]byte{"/encoder.onnx": getEncoderBytes()})
	fetch, files, _ := newSpeechFetch(t, server, false, server.URL+"/encoder.onnx")
	stored, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)
	if err != nil {
		t.Fatal(err)
	}
	files.cfg.Offline = true

	path, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)

	if err != nil || path != stored {
		t.Errorf("offline fetch = %q, %v, want %q", path, err, stored)
	}
	if got := server.getRequests(); len(got) != 1 {
		t.Errorf("requests = %v, want the first fetch only", got)
	}
}

type titledStatusLog struct {
	mu       sync.Mutex
	statuses []plugin.DependencyStatus
	titles   []string
}

func (l *titledStatusLog) add(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.statuses = append(l.statuses, status)
	l.titles = append(l.titles, req.GetTitle())
}

func (l *titledStatusLog) getLast() (string, plugin.DependencyStatus) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.statuses) == 0 {
		return "", plugin.DependencyStatus{}
	}
	return l.titles[len(l.titles)-1], l.statuses[len(l.statuses)-1]
}

func TestADownloadStatusNamesTheModelTitleAndDeclaredSize(t *testing.T) {
	sized, stop := make(chan struct{}), make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-sized:
		case <-stop:
			return
		case <-r.Context().Done():
			return
		}
		half := len(getEncoderBytes()) / 2
		for _, part := range [][]byte{getEncoderBytes()[:half], getEncoderBytes()[half:]} {
			if _, err := w.Write(part); err != nil {
				t.Error(err)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(stop) })
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	statuses := &titledStatusLog{}
	var once sync.Once
	registry := NewObservedRegistry(func(req manifest.DependencyRequirement, status plugin.DependencyStatus) {
		statuses.add(req, status)
		if status.Message == "Downloading "+encoderTitle && status.BytesTotal == int64(len(getEncoderBytes())) {
			once.Do(func() { close(sized) })
		}
	}, files)
	manifests := declaredManifests{speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel(speechEncoder, encoderTitle, getEncoderBytes(), server.URL+"/encoder.onnx"))}
	fetch := newDeclaredFetch(manifests, registry)
	ctx, cancel := context.WithTimeout(t.Context(), declaredWait)
	defer cancel()

	path, err := fetch.Execute(ctx, speechPlugin, speechEncoder)

	if err != nil {
		t.Fatalf("fetch = %v: no status named %q with the declared size %d before the response", err, "Downloading "+encoderTitle, len(getEncoderBytes()))
	}
	if got := readModelFile(t, path); !bytes.Equal(got, getEncoderBytes()) {
		t.Errorf("the file holds %q", got)
	}
	title, last := statuses.getLast()
	if title != encoderTitle || last.State != plugin.DependencyStateReady {
		t.Errorf("the last status is %q %+v, want %q ready", title, last, encoderTitle)
	}
}

func TestCancelingTheDownloadOfAModelEndsTheRequestAsCanceled(t *testing.T) {
	arrived := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		arrived <- struct{}{}
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	files := newTestProvisioner(t, false)
	files.cfg.Client = server.Client()
	registry := NewObservedRegistry(nil, files)
	manifests := declaredManifests{speechPlugin: readDeclaredManifest(t, speechPlugin, declareModel(speechEncoder, encoderTitle, getEncoderBytes(), server.URL+"/encoder.onnx"))}
	fetch := newDeclaredFetch(manifests, registry)
	answer := make(chan error, 1)
	go func() {
		_, err := fetch.Execute(t.Context(), speechPlugin, speechEncoder)
		answer <- err
	}()
	select {
	case <-arrived:
	case <-time.After(declaredWait):
		t.Fatal("no download started")
	}

	registry.CancelAll()

	select {
	case err := <-answer:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the request ended with %v, want the cancellation", err)
		}
	case <-time.After(declaredWait):
		t.Fatal("the request did not end after the cancel")
	}
}
