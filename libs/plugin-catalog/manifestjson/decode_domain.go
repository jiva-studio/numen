package manifestjson

import "github.com/jiva-studio/numen/libs/notes/domain"

func newContentTypes(texts []string) []domain.ContentType {
	var out []domain.ContentType
	for _, text := range texts {
		out = append(out, domain.ContentType(text))
	}
	return out
}

func newHandlerRank(text string) domain.HandlerRank {
	return domain.HandlerRank(text)
}
