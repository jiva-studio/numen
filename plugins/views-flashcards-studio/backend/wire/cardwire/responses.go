package cardwire

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/filewire"
)

// GetLimit is how many card models a request for limit of them is answered with.
func GetLimit(limit int32) int {
	if limit <= 0 || limit > MaxCardModels {
		return MaxCardModels
	}
	return int(limit)
}

// NewListResponse is the card models read, and the total the vault holds.
func NewListResponse(found []cards.CardModelSummary, total int, err error) (*api.ListCardModelsResponse, error) {
	if err != nil {
		return nil, apierror.NewInternal(err)
	}
	summaries := make([]*api.CardModelSummary, 0, len(found))
	for _, cardModel := range found {
		summaries = append(summaries, NewWireSummary(cardModel))
	}
	return &api.ListCardModelsResponse{CardModels: summaries, Total: int32(total)}, nil
}

// NewReadCardModelResponse answers the read of a card model. The title is looked up only for a card
// model that was read.
func NewReadCardModelResponse(
	found cards.CardModelContents, err error, title func(path string) string,
) (*api.ReadCardModelResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.ReadCardModelResponse{CardModel: nil, Error: api.ErrorCodeUnspecified, At: nil}
	if code, ok := apierror.GetCardModelError(found.Outcome, found.Type); ok {
		out.Error = code
		return out, nil
	}
	out.CardModel = NewWireCardModel(found.Path, title(found.Path), found.Body)
	// What the file was when this came out of it, for the client to present
	// when it writes the card model back.
	out.At = filewire.NewStat(found.FileStat)
	return out, nil
}

// NewReadDeckResponse answers the read of a deck. The title is looked up only for a deck that was
// read.
func NewReadDeckResponse(found cards.DeckContents, err error, title func(path string) string) (*api.ReadDeckResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.ReadDeckResponse{Deck: nil, Error: api.ErrorCodeUnspecified, At: nil, Bound: 0}
	if code, ok := apierror.GetDeckError(found.Outcome, found.Type); ok {
		out.Error = code
		// The bound is sent with the error code, so the interface names it
		// without storing a number of its own.
		if code == api.ErrorCodeDeckTooLarge {
			out.Bound = cards.MaxBytes
		}
		return out, nil
	}
	out.Deck = NewWireDeck(found.Path, title(found.Path), found.Body, found.CardModels)
	out.At = filewire.NewStat(found.FileStat)
	return out, nil
}

// NewWriteDeckResponse answers the end of the write of a deck.
func NewWriteDeckResponse(pos domain.FileStat, err error) (*api.WriteDeckResponse, error) {
	out, err := filewire.ConvertWrite(pos, err, filewire.Reason{Err: note.ErrTooLarge, Code: api.ErrorCodeDeckTooLarge})
	if err != nil {
		return nil, err
	}
	res := &api.WriteDeckResponse{Error: out.Error, At: out.At, Bound: 0, IsIndexOutdated: out.IsIndexOutdated}
	if out.Error == api.ErrorCodeDeckTooLarge {
		res.Bound = cards.MaxBytes
	}
	return res, nil
}

func NewWriteCardModelResponse(pos domain.FileStat, err error) (*api.WriteCardModelResponse, error) {
	out, err := filewire.ConvertWrite(pos, err)
	if err != nil {
		return nil, err
	}
	return &api.WriteCardModelResponse{Error: out.Error, At: out.At, IsIndexOutdated: out.IsIndexOutdated}, nil
}

// ConvertRename answers the end of a rename of a field.
func ConvertRename(renamed cards.RenameResult, err error) (*api.RenameCardModelFieldResponse, error) {
	if err == nil || errors.Is(err, note.ErrIndexOutdated) {
		out := NewRenameResponse(renamed, filewire.NewStat(renamed.CardModel))
		out.IsIndexOutdated = apierror.IsIndexOutdated(err)
		return out, nil
	}
	// The name a rename is given is the client's: one the card model does not
	// declare, and one it already declares, are both a name to correct.
	if errors.Is(err, format.ErrNoSuchField) || errors.Is(err, format.ErrFieldTaken) {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	reason, ok := apierror.GetErrorCode(err)
	if !ok {
		return nil, apierror.NewError(apierror.GetCode(err), err)
	}
	return &api.RenameCardModelFieldResponse{
		Decks: nil, Cards: 0, NotWritten: nil, Error: reason, At: nil, IsIndexOutdated: false,
	}, nil
}
