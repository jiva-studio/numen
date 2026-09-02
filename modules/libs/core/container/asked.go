package container

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// asked is the sources a person named, in the order they named them. A run
// empties it before it takes up anything nobody asked for, and a source named
// twice waits in it once.
//
// Nothing here is filtered. A person naming a file by hand has said that this
// file is worth the machine's time, so the size a queue leaves alone and an
// installation that listens to nothing on its own both hold no sway here.
//
// It locks nothing: it is held by the run that owns it, under that run's lock.
type asked struct {
	line []wanted
	held map[string]bool
}

// wanted is one source a person named: the vault it is in, and where.
type wanted struct {
	vault domain.Vault
	path  string
}

// want puts a source at the back of the line and says whether it went in.
func (a *asked) want(v domain.Vault, path string) bool {
	if a.held == nil {
		a.held = map[string]bool{}
	}
	key := named(v, path)
	if a.held[key] {
		return false
	}
	a.held[key] = true
	a.line = append(a.line, wanted{vault: v, path: path})
	return true
}

// take is the source at the front of the line, and false where nobody is
// waiting.
func (a *asked) take() (wanted, bool) {
	if len(a.line) == 0 {
		return wanted{}, false
	}
	one := a.line[0]
	a.line = a.line[1:]
	delete(a.held, named(one.vault, one.path))
	return one, true
}

// waiting is how many sources are named and not yet taken up.
func (a *asked) waiting() int { return len(a.line) }
