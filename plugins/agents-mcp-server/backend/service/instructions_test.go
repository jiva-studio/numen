package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestTheServerTellsTheAgentHowToCiteALibraryPassage(t *testing.T) {
	t.Parallel()
	hostClient := &fakeToolClient{tools: []*tools.ToolInfo{
		{Name: "library_search", Description: "Search the library", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly},
		{Name: "library_read", Description: "Read the library", InputSchema: []byte(`{"type":"object"}`), Effect: tools.ReadOnly},
	}}
	srv, err := NewServer(Config{Addr: "127.0.0.1:0", Token: "test-token"}, hostClient)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := srv.Close(t.Context()); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	session := connectStreamableClient(ctx, t, srv.GetHTTPURL(), "test-token")
	defer cleanup.Close(t, session)

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, listedTool := range listed.Tools {
		names = append(names, listedTool.Name)
	}
	if strings.Join(names, ",") != "library_read,library_search" && strings.Join(names, ",") != "library_search,library_read" {
		t.Errorf("listed %v, want the two library tools", names)
	}

	instructions := session.InitializeResult().Instructions
	for _, want := range []string{
		"library_search",
		"`at`",
		"[the Remuna episode](numen:library%2FA%20Book.pdf?start=62690&length=1246)",
		"markdown link",
		"source_focus",
		"[[Harmonic oscillator]]",
		"file://",
		"library_read",
		"mid-sentence",
	} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not say %q:\n%s", want, instructions)
		}
	}
}
