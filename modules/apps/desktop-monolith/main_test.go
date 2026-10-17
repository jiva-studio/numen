package main

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

type recordingRegistry struct {
	registered map[string]any
}

func (r *recordingRegistry) RegisterDocumentHandler(h numenv1connect.DocumentServiceHandler) {
	r.registered["document"] = h
}

func (r *recordingRegistry) RegisterBookHandler(h numenv1connect.BookServiceHandler) {
	r.registered["book"] = h
}

func TestWireMonolithHandlersRegistersOnlyRealHandlers(t *testing.T) {
	r := &recordingRegistry{registered: map[string]any{}}

	wireMonolithHandlers(r)

	got := make([]string, 0, len(r.registered))
	for name := range r.registered {
		got = append(got, name)
	}
	sort.Strings(got)
	if want := []string{"book", "document"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("registered %v, want %v", got, want)
	}
	for name, h := range r.registered {
		if h == nil {
			t.Errorf("%s handler is nil", name)
		}
	}
}

func TestHandlerRegistryHasNoStubCapabilities(t *testing.T) {
	typ := reflect.TypeOf((*handlerRegistry)(nil)).Elem()
	for _, absent := range []string{"RegisterEmbeddingHandler", "RegisterRecognitionHandler", "RegisterTranscriptionHandler"} {
		if _, ok := typ.MethodByName(absent); ok {
			t.Errorf("monolith registry exposes %s; a capability without a real handler stays absent", absent)
		}
	}
}
