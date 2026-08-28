package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// A stencil shows a card once through each face it carries, and a face is not
// addressed by its name here. Two faces of one name are two faces, and what
// each holds is in the file.
func TestAStencilKeepsEveryFaceItIsGiven(t *testing.T) {
	body, err := container.StencilBody([]format.Face{
		{Name: "Recognise", Front: "{{Name}}", Back: "{{Height}}"},
		{Name: "Recognise", Front: "{{Height}}", Back: "{{Name}}"},
	})
	if err != nil {
		t.Fatalf("stencil body: %v", err)
	}

	read := format.ReadStencil(domain.Note{Body: body})
	if len(read.Faces) != 2 {
		t.Fatalf("faces = %+v, want both of them", read.Faces)
	}
	if read.Faces[0].Front != "{{Name}}" || read.Faces[0].Back != "{{Height}}" {
		t.Errorf("the first face = %+v", read.Faces[0])
	}
	if read.Faces[1].Front != "{{Height}}" || read.Faces[1].Back != "{{Name}}" {
		t.Errorf("the second face = %+v", read.Faces[1])
	}
}
