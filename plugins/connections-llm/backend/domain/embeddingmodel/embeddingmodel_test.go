package embeddingmodel

import "testing"

func TestAModelOfTheTableIsFoundByEitherNameIgnoringCase(t *testing.T) {
	for _, name := range []string{"text-embedding-3-small", "OpenAI/Text-Embedding-3-Small", "  text-embedding-3-small "} {
		profile, ok := Find(name)
		if !ok || profile.Dimensions != 1536 || profile.MaxTokens != 8191 || profile.Pooling != PoolingCLS {
			t.Errorf("Find(%q) = %+v, %v", name, profile, ok)
		}
	}
}

func TestAModelOutsideTheTableHasNoProfile(t *testing.T) {
	if _, ok := Find("nomic-embed-text"); ok {
		t.Error("a model outside the table has a profile")
	}
}
