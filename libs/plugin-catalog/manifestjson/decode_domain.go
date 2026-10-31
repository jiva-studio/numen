package manifestjson

import "github.com/jiva-studio/numen/libs/notes/domain"

func newContentType(text string) domain.ContentType {
	return domain.ContentType(text)
}

func newContentTypes(texts []string) []domain.ContentType {
	if texts == nil {
		return nil
	}
	out := make([]domain.ContentType, 0, len(texts))
	for _, text := range texts {
		out = append(out, domain.ContentType(text))
	}
	return out
}

func newTextKinds(texts []string) []domain.TextKind {
	if texts == nil {
		return nil
	}
	out := make([]domain.TextKind, 0, len(texts))
	for _, text := range texts {
		out = append(out, domain.TextKind(text))
	}
	return out
}

func newHandlerRank(text string) domain.HandlerRank {
	return domain.HandlerRank(text)
}
