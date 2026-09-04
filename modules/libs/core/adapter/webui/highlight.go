package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
)

// errNoHighlight is what a build with nothing to place a passage with answers.
var errNoHighlight = errors.New("this build cannot say where a passage is")

// covering is what the window is told the runs of the prose cover, one entry per
// run and in the order they were asked about.
type covering struct {
	Runs []covered `json:"runs"`
}

// covered is what one run covers.
type covered struct {
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
	if a.Highlight == nil {
		http.Error(w, errNoHighlight.Error(), http.StatusNotImplemented)
		return
	}
	showing := a.Showing()
	if showing.ID == "" {
		refuse(w, errNoVault)
		return
	}
	runs, err := places(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// A reading is read off the disk, and a layer is asked of the document,
	// which waits for one of the pool. A window is answered or told to ask
	// again; it is not held while a recognition has every worker.
	ctx, cancel := context.WithTimeout(r.Context(), patience)
	defer cancel()

	found, err := a.Highlight.Execute(ctx, showing, path, runs)
	if err != nil {
		refuse(w, err)
		return
	}

	told := covering{Runs: make([]covered, 0, len(found))}
	for _, pages := range found {
		one := covered{Marks: make([]onPage, 0, len(pages))}
		for _, page := range pages {
			marks := onPage{Page: page.Index, Rects: make([]rect, 0, len(page.Rects))}
			for _, box := range page.Rects {
				marks.Rects = append(marks.Rects, rect{
					MinX: box.MinX, MinY: box.MinY, MaxX: box.MaxX, MaxY: box.MaxY,
				})
			}
			one.Marks = append(one.Marks, marks)
		}
		told.Runs = append(told.Runs, one)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(told)
}

// longestRun is the most text one question about a place may cover. A passage
// is a few hundred characters; a run of a million asks where the whole book is,
// one page at a time.
const longestRun = 100_000

// places is which parts of the source's text the window is asking about: a
// `start` and a `length` for each of them, paired in the order they are given.
func places(query url.Values) ([]highlight.Stretch, error) {
	starts, lengths := query["start"], query["length"]
	if len(starts) != len(lengths) {
		return nil, fmt.Errorf("%d places begin and %d have a length", len(starts), len(lengths))
	}
	if len(starts) == 0 || len(starts) > domain.MostLit {
		return nil, fmt.Errorf("ask about between one and %d places, not %d", domain.MostLit, len(starts))
	}
	runs := make([]highlight.Stretch, 0, len(starts))
	for i, at := range starts {
		start, err := strconv.Atoi(at)
		if err != nil || start < 0 {
			return nil, fmt.Errorf("start: %q is not a place in the text", at)
		}
		length, err := strconv.Atoi(lengths[i])
		if err != nil || length < 1 || length > longestRun {
			return nil, fmt.Errorf("length: %q is not a run of the text", lengths[i])
		}
		runs = append(runs, highlight.Stretch{Start: start, Length: length})
	}
	return runs, nil
}
