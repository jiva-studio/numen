package flashcards

import (
	"sync"
	"testing"
)

// How many places of a curve run at once is what the build was told, and a
// build told nothing runs them one at a time. Reading the machine's cores here
// instead would answer whatever the machine happens to have.
func TestHowManyPlacesRunAtOnce(t *testing.T) {
	t.Parallel()
	for _, cores := range []int{0, 1, 3} {
		want := max(1, cores)
		// A place holds its ground until as many as are admitted stand beside
		// it, so what stood there at once is what the room allowed.
		let := make(chan struct{})
		var letting sync.Once
		var mu sync.Mutex
		running, most := 0, 0

		err := (ProjectCurve{Cores: cores}).places(2*Points, func(int) error {
			mu.Lock()
			running++
			if running > most {
				most = running
			}
			full := running == want
			mu.Unlock()
			if full {
				letting.Do(func() { close(let) })
			}
			<-let
			mu.Lock()
			running--
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if most != want {
			t.Errorf("told %d cores, %d places ran at once, want %d", cores, most, want)
		}
	}
}
