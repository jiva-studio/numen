# The core — what holds here

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file is the core's own, and it holds over `modules/libs/core`.

The architecture is in [`docs/adr/0004-a-hexagonal-core-in-go.md`](../../../docs/adr/0004-a-hexagonal-core-in-go.md)
and [`docs/adr/0035-where-a-port-is-declared-and-where-an-adapter-stands.md`](../../../docs/adr/0035-where-a-port-is-declared-and-where-an-adapter-stands.md).
Read those for the shape. What follows is the rules a test refuses, and the
test that refuses each — none of them is carried in anybody's memory.

## The layers, and what a package may reach

`container/layers_test.go` and `container/direction_test.go` hold these. Each
carries a table of what it refuses and a guard against passing vacuously.

| The rule | The test |
|---|---|
| A package stands on the layer its path says, and reaches only downward | `TestTheLayersAreWhatTheyAre` |
| Nothing the core is compiled from reaches an adapter | `TestNothingTheCoreIsCompiledFromReachesOutward` |
| A pure package is tested without an adapter | `TestNoPurePackageIsTestedThroughAnAdapter` |
| An adapter does not name the port it answers | `TestNoAdapterNamesThePortItSatisfies` |
| The composition root hands out no adapter type | `TestTheRootHandsOutNoAdapterTheCompilerHolds` |
| A package is imported under the name its layer gives it | `TestNoPackageIsImportedUnderItsLayer` |

Both rules carry a baseline of the edges this installation still has, and
`container/baseline_test.go` refuses an entry naming an edge nothing reaches any
more. A baseline only shrinks.

## What the core may not do

| The rule | The test |
|---|---|
| Nothing reaches the machine: no `os`, no filesystem, no network | `TestNothingOfTheCoreReachesTheMachine` |
| Nothing reads the machine's clock — an instant arrives as an argument | `TestNothingOfTheCoreReadsTheMachinesClock` |
| Nothing logs ([ADR 0037](../../../docs/adr/0037-nothing-is-logged.md)) | `TestNothingOfTheCoreLogs` |

## Ports

A port is declared where it is consumed, and every method on one is called from
outside `port/`.

| The rule | The test |
|---|---|
| Every port is asked for somewhere else | `TestEveryPortIsAskedForSomewhereElse` |
| Every type a port declares is named somewhere else | `TestEveryTypePortDeclaresIsNamedSomewhereElse` |
| Every port method is called from outside `port/` | `TestEveryPortMethodIsCalledSomewhereElse` |
| Every adapter serving the schema is a driving one | `TestEveryAdapterServingTheSchemaIsDriving` |

## Names the core carries

| The rule | The test |
|---|---|
| An identity is a word of the core's own, in a field and in a signature alike | `TestTheCoresIdentitiesAreItsOwn`, `TestTheIdentitiesAPortAsksForAreTheCoresOwn` |
| An instant is a `time.Time` | `TestEveryInstantTheCoreCarriesIsATime` |
| No type is named by a gerund or a participle | `TestNoTypeOfTheCoreIsNamedByAGerundOrParticiple` |
| Every word of the glossary names a type | `TestEveryWordOfTheDictionaryNamesAType` |

The vocabulary itself is [`docs/glossary.md`](../../../docs/glossary.md), and
its preamble is what a rename is judged against: a concept takes the word its
field already gives it, a word means one thing inside its context, and nothing
is coined or suffixed to keep clear of a name another context uses.

## Constructors

Every package has `New*` functions. A dependency is given at construction and
is not assignable afterwards; a guard reading `if u.X == nil` in a method is
the residue of a field that still is, and it goes when the field is closed.

## Where a rule lives

A rule about one value is a method on that value. A rule over several is a
function in the package that owns the subject. `domain/` holds what is true of
a note or a card; the arithmetic over a schedule is `flashcards/review`, and
the format is `markdown`.

## Writing a new rule

A rule is three things, and a rule missing any of them is not landed:

1. the walk over the tree, with a guard that fails when it reads nothing;
2. a negative control — a fixture written to be refused, asserting exactly what
   the rule catches and what it lets through;
3. a baseline, where the tree does not hold yet, that only shrinks.
