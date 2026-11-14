package embeddingmodel

import "testing"

func TestAModelOfTheTableIsFoundByEitherNameIgnoringCase(t *testing.T) {
	for _, tc := range []struct {
		names     []string
		dims, max int
	}{
		{[]string{"text-embedding-3-small", "OpenAI/Text-Embedding-3-Small", "  text-embedding-3-small "}, 1536, 8191},
		{[]string{"text-embedding-3-large", "openai/text-embedding-3-large"}, 3072, 8191},
		{[]string{"text-embedding-ada-002", "openai/text-embedding-ada-002"}, 1536, 8191},
		{[]string{"baai/bge-m3"}, 1024, 8192},
		{[]string{"qwen/qwen3-embedding-8b"}, 4096, 32768},
	} {
		for _, name := range tc.names {
			profile, ok := Find(name)
			if !ok || profile.Dimensions != tc.dims || profile.MaxTokens != tc.max || profile.Pooling != PoolingCLS {
				t.Errorf("Find(%q) = %+v, %v, want dims=%d max=%d", name, profile, ok, tc.dims, tc.max)
			}
		}
	}
}

func TestAModelOutsideTheTableHasNoProfile(t *testing.T) {
	if _, ok := Find("nomic-embed-text"); ok {
		t.Error("a model outside the table has a profile")
	}
}
