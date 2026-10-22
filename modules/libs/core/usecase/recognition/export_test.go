package recognition

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// NewMemoryRecognise is a Recognise over the given vault readers and page
// renderer, writing to an in-memory store and reading with a model that says
// one line a page. onPage is called as each page is shown to the model. The
// second answer is the hash the source at a path was recorded under.
func NewMemoryRecognise(
	readers port.VaultReaders,
	documents port.PageRenderer,
	onPage func(),
) (Recognise, func(v domain.VaultID, path string) string) {
	index := newStore()
	model := &speaker{says: "A page.", stop: func(int) {
		if onPage != nil {
			onPage()
		}
	}}
	r := NewRecognise(readers, index, fakeStores{newShelf()}, documents, model)
	r.Batch = 1
	return r, func(v domain.VaultID, path string) string { return index.sources[v][path].Hash }
}
