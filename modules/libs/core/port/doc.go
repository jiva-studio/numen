// Package port declares what the core needs from the world outside it.
//
// Every interface is small and named after the need it fills: the core asks
// for somewhere to read a vault from. Adapters are named after the technology,
// which is the only place it is allowed to appear.
//
// Writing and reading are separate. A repository is a collection of aggregates —
// put one in, take one out, remove one — and nothing else. Anything that answers
// a question across many of them, in a shape that is not an aggregate, is a
// query: matches and counts are read models, and asking a repository for them
// turns it into a service wearing a repository's name.
package port
