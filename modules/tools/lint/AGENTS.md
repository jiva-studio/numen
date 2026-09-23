# The linter — what holds here

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file is the linter's own, and it holds over
`modules/tools/lint`.

Each rule is a pair: `<rule>.mjs` holds the walk and `<rule>.test.mjs` holds
what it refuses. `npm run check` runs every one of them, and `make lint` runs
that.

## A rule is three things

A rule missing any of them is not landed.

1. **The walk**, over the sources `source.mjs` hands it, with a guard that
   fails when it read nothing. A rule checked against an empty list passes, and
   a rule that passes for that reason is worse than no rule.
2. **The negative control** — a fixture written to be refused, asserting
   exactly what the rule catches and what it lets through. Both halves: a rule
   nobody has watched let something through is a rule nobody can trust to.
3. **The baseline**, where the tree does not hold yet: the entries still
   standing, each under the reason it stands. A test refuses an entry naming
   something that is no longer there, so the list only shrinks.

## A baseline entry is a border, not a plea

An entry says why the name it holds cannot be moved: a field matched against a
generated message by its shape, a word a settings file writes, an attribute the
browser named. A name that is merely inconvenient to change is not an entry.

## What the rules hold today

| The rule | Refuses |
|---|---|
| `booleans.mjs` | a field holding a boolean whose name does not say so |
| `nouns.mjs` | a type named by a gerund or a participle |
| `verbs.mjs` | a function named by anything but an imperative verb phrase |
| `composables.mjs` | a composable not named `use<Feature>` |
| `filenames.mjs` | a file or folder outside the casing its kind takes |
| `aliases.mjs`, `reach.mjs`, `reaching.mjs` | an import that does not say which layer it reaches |
| `catches.mjs` | a catch that swallows what it caught |
| `exports.mjs` | a barrel collision |
| `quotes.mjs` | a quotation mark written as the wrong glyph |
| `refs.mjs` | a template ref asked for that no template binds |
| `styles.mjs` | a style rule nothing can reach |

The class no machine checks is a name that is grammatical and still says
nothing: a bare noun where a question was meant, an adjective standing for a
state, a metaphor where an engineering term exists. That one is read by a
person, and the gatekeeper stage of the review skill carries the checklist.
