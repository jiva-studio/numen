package searchquery

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (u *ApplyStages) Execute(
	ctx context.Context, v domain.Vault, q searchquery.Query, r searchquery.SearchResults, priority modelcall.Priority,
) (searchquery.SearchResults, error) {
	for _, stage := range q.Stages {
		var err error
		switch stage.Kind {
		case searchquery.StageRerank:
			r, err = u.rerank(ctx, v, r, stage.Rerank, priority)
		case searchquery.StageTop:
			r = searchquery.ApplyTop(r, stage.Count)
		case searchquery.StageSortBy:
			r = searchquery.ApplySort(r, stage.Sort)
		}
		if err != nil {
			return searchquery.SearchResults{}, err
		}
	}
	return r, nil
}

func (u *ApplyStages) rerank(
	ctx context.Context, v domain.Vault, r searchquery.SearchResults, s searchquery.RerankStage, priority modelcall.Priority,
) (searchquery.SearchResults, error) {
	limits := u.reranker.limits
	window := s.Window
	if window == 0 {
		window = limits.RerankWindow
	}
	units := searchquery.SelectRerankUnits(r, window, limits.MaxReranked)
	if len(units) == 0 {
		return r, nil
	}
	texts, err := u.readUnitTexts(ctx, v, r, units)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	scores, report, err := u.reranker.score(ctx, rerankInput{query: s.Query, instruction: s.Instruction, texts: texts, priority: priority})
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	reranked := searchquery.ApplyRerank(r, window, units, scores)
	reranked.Reranks = append(reranked.Reranks, report)
	return reranked, nil
}

func (u *ApplyStages) readUnitTexts(
	ctx context.Context, v domain.Vault, r searchquery.SearchResults, units []searchquery.RerankUnit,
) ([]string, error) {
	files, err := u.openFiles(v)
	if err != nil {
		return nil, err
	}
	most := u.reranker.limits.MaxRerankedBytes
	out := make([]string, len(units))
	for i, unit := range units {
		result := r.Results[unit.ResultIndex]
		text, err := files.read(ctx, result)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", result.Document.Path, err)
		}
		if unit.ChunkIndex < 0 {
			out[i] = searchquery.TruncateText(result.Document.Title+"\n"+getWholeText(result, text), most)
			continue
		}
		c := result.Chunks[unit.ChunkIndex]
		out[i] = searchquery.TruncateText(getSpan(text.prose, c.Start, c.Length), most)
	}
	return out, nil
}

func getWholeText(result searchquery.SearchResult, text fileText) string {
	if text.isRead {
		return text.prose[getProseStart(text.prose):]
	}
	return result.Text
}

// unitFiles reads the text of each file once.
type unitFiles struct {
	readFile func(ctx context.Context, path string) ([]byte, error)
	texts    textdomain.Texts
	cache    map[string]fileText
}

func (u *ApplyStages) openFiles(v domain.Vault) (*unitFiles, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, err
	}
	files := &unitFiles{readFile: reader.Read, cache: map[string]fileText{}}
	if u.texts != nil {
		if files.texts, err = u.texts.Open(v); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (f *unitFiles) read(ctx context.Context, result searchquery.SearchResult) (fileText, error) {
	if result.IsWholeFile() && result.Document.SourceKind != domain.KindNote {
		return fileText{}, nil
	}
	key := result.Document.Path + "\x00" + result.TextVersion
	if text, found := f.cache[key]; found {
		return text, nil
	}
	text, err := readText(ctx, f.readFile, f.texts, result.Document.Path, result.TextVersion)
	if err != nil {
		return fileText{}, err
	}
	f.cache[key] = text
	return text, nil
}
