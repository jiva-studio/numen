package cardwire

import "testing"

func TestGetLimitKeepsARequestWithinWhatOneResponseCarries(t *testing.T) {
	for _, one := range []struct {
		limit int32
		want  int
	}{
		{0, MaxCardModels},
		{-1, MaxCardModels},
		{1, 1},
		{MaxCardModels, MaxCardModels},
		{MaxCardModels + 1, MaxCardModels},
	} {
		if got := GetLimit(one.limit); got != one.want {
			t.Errorf("a request for %d is answered with %d, want %d", one.limit, got, one.want)
		}
	}
}
