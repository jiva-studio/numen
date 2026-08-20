package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// errNoMarking is what a build with nothing to place a passage with answers.
var errNoMarking = errors.New("this build cannot say where a passage is")

// lit is what the window is told a run of the prose covers.
type lit struct {
	Marks []onPage `json:"marks"`
}

// onPage is one page and what to light on it.
type onPage struct {
	Page  int    `json:"page"`
	Rects []rect `json:"rects"`
}

// rect is a place on a page, in fractions of it, so a page drawn at any size
// lines up by multiplying.
type rect struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
}

// Marks answers where a run of a source's text sits: the pages it falls on and,
// on each, the rectangles covering it.
func (a *API) Marks(w http.ResponseWriter, r *http.Request, path string) {
	if a.Marking == nil {
		http.Error(w, errNoMarking.Error(), http.StatusNotImplemented)
		return
	}
	start, length, err := run(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// A reading is read off the disk, and a layer is asked of the document,
	// which waits for one of the pool. A window is answered or told to ask
	// again; it is not held while a recognition has every worker.
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	found, err := a.Marking.Execute(ctx, a.Vault, path, start, length)
	if err != nil {
		refuse(w, err)
		return
	}

	told := lit{Marks: make([]onPage, 0, len(found))}
	for _, page := range found {
		one := onPage{Page: page.Page, Rects: make([]rect, 0, len(page.Rects))}
		for _, box := range page.Rects {
			one.Rects = append(one.Rects, rect{
				MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY,
			})
		}
		told.Marks = append(told.Marks, one)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(told)
}

// run is which part of the source's text the window is asking about: where it
// begins, and how many bytes of it there are.
// longestRun is the most text one question about a place may cover. A passage
// is a few hundred characters; a run of a million asks where the whole book is,
// one page at a time.
const longestRun = 100_000

func run(query url.Values) (start, length int, err error) {
	start, err = strconv.Atoi(query.Get("start"))
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("start: %q is not a place in the text", query.Get("start"))
	}
	length, err = strconv.Atoi(query.Get("length"))
	if err != nil || length < 1 || length > longestRun {
		return 0, 0, fmt.Errorf("length: %q is not a run of the text", query.Get("length"))
	}
	return start, length, nil
}
