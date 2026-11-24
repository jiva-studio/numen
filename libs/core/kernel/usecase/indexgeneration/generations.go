// Package indexgeneration counts the changes of the index of each vault in memory, under an epoch that
// differs between two starts of the core.
package indexgeneration

import (
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Generations is the count of index changes of each vault. It is safe to use from several goroutines.
type Generations struct {
	epoch  string
	mu     sync.Mutex
	counts map[domain.VaultID]int
}

// New returns the generations of a start of the core named by epoch.
func New(epoch string) *Generations {
	return &Generations{epoch: epoch, counts: map[domain.VaultID]int{}}
}

func (g *Generations) GetGeneration(v domain.VaultID) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.epoch + "." + strconv.Itoa(g.counts[v])
}

// BumpGeneration records one change of the index of the vault.
func (g *Generations) BumpGeneration(v domain.VaultID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counts[v]++
}
