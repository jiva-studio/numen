package cardwire

import "testing"

func TestGetLimitKeepsARequestWithinWhatOneResponseCarries(t *testing.T) {
	for _, one := range []struct {
		asked int32
		want  int
	}{
		{0, MaxCardModels},
		{-1, MaxCardModels},
		{1, 1},
		{MaxCardModels, MaxCardModels},
		{MaxCardModels + 1, MaxCardModels},
	} {
		if got := GetLimit(one.asked); got != one.want {
			t.Errorf("a request for %d is answered with %d, want %d", one.asked, got, one.want)
		}
	}
}
