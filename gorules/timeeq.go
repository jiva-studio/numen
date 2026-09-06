//go:build ruleguard

// Rules gocritic's ruleguard checker runs, over every Go module here.
//
// The tag keeps this out of every build; ruleguard reads the file by the path
// `.golangci.yml` names and pays no attention to it.
//
// Taken from dgryski/semgrep-go's `timeeq` rule, under its MIT licence.
package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// timeeq refuses a time.Time compared with an operator instead of Equal.
//
// A time.Time carries a location and, when a clock handed it out, a monotonic
// reading. `==` compares both, so two values naming one instant compare unequal
// as soon as one of them has been round-tripped or read back in another zone.
// domain.Fingerprint.ModTime was written this way and every file in a vault
// read as changed.
//
// A map keyed by one is the same fault spread over a lookup: the key that goes
// in is not the key that comes back out.
func timeeq(m dsl.Matcher) {
	m.Match("$t0 == $t1").Where(m["t0"].Type.Is("time.Time")).
		Report("a time.Time compares its location and its monotonic reading — use Equal")
	m.Match("$t0 != $t1").Where(m["t0"].Type.Is("time.Time")).
		Report("a time.Time compares its location and its monotonic reading — use Equal")
	m.Match(`map[$k]$v`).Where(m["k"].Type.Is("time.Time")).
		Report("a map keyed by time.Time hands back nothing for a key naming the same instant")
}
