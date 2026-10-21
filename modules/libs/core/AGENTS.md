# The core — architecture and invariants

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file defines specific rules for `modules/libs/core`.

The architecture follows [`.agents/rules/architecture.md`](../../../.agents/rules/architecture.md)
("Ports and adapters"). The following table summarizes architectural invariants
and the corresponding tests enforcing them.

## Layer boundaries and dependency direction

`container/layers_test.go` and `container/direction_test.go` enforce these boundaries.
Each test verifies adherence to structural rules.

| The rule | The test |
|---|---|
| A package stands on the layer its path says, and reaches only downward | `TestTheLayersAreWhatTheyAre` |
| Nothing the core is compiled from reaches an adapter | `TestNothingTheCoreIsCompiledFromReachesOutward` |
| A pure package is tested without an adapter | `TestNoPurePackageIsTestedThroughAnAdapter` |
| A use case builds only on the use cases `useCaseEdges` declares for it | `TestUseCaseImportsFollowDeclaredEdges` |
| An adapter does not name the port it answers | `TestNoAdapterNamesThePortItSatisfies` |
| The composition root hands out no adapter type | `TestTheRootHandsOutNoAdapterTheCompilerHolds` |
| A package is imported under the name its layer gives it | `TestNoPackageIsImportedUnderItsLayer` |

## What the core may not do

| The rule | The test |
|---|---|
| Nothing reaches the machine: no `os`, no filesystem, no network | `TestNothingOfTheCoreReachesTheMachine` |
| Nothing reads the machine's clock — an instant arrives as an argument | `TestNothingOfTheCoreReadsTheMachinesClock` |
| Nothing logs | `TestNothingOfTheCoreLogs` |

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
| A driving adapter calls no port outside its wiring files, and is on the depguard rule | `TestNoDrivingAdapterCallsAPortOutsideItsWiring`, `TestEveryDrivingAdapterIsOnTheLintRule` |
| The core arms no timer of the machine's | `TestNothingOfTheCoreReadsTheMachinesClock` |
| The core probes no value for another interface | `TestNoCodeOfTheCoreProbesAValueForAnotherInterface` |
| A port names no type from `internal/` | `TestNoPortNamesATypeFromInternal` |

No type is named by a gerund or a participle.

The vocabulary itself is [`docs/glossary.md`](../../../docs/glossary.md), and
its preamble is what a rename is judged against: a concept takes the word its
field already gives it, a word means one thing inside its context, and nothing
is coined or suffixed to keep clear of a name another context uses.

## Constructors

A use case is made with everything it needs, or it is not made. The constructor
takes the collaborators, refuses what it cannot make, and the fields holding
them are unexported — so no half-built one exists for a method to check.

| The rule | The test |
|---|---|
| A use case does not carry a collaborator its constructor takes in an exported field | `TestNoUseCaseCarriesACollaboratorInTheOpen` |

No use case exposes a collaborator.

A dependency that may genuinely be absent is not a nil left in a field: it is a
case the composition root decides explicitly — `ProofreadingConfig.Recognition`
returns a boolean flag (e.g. `isConfigured`), so a window that cannot proofread never gets something to
proofread with.

A policy is not a collaborator. How many pages a batch holds, where progress is
reported, what to do after a write — those are set on the made value and carry
a default.

## Where a rule lives

A rule about one value is a method on that value. A rule over several is a
function in the package that owns the subject. `domain/` holds what is true of
a note or a card; the arithmetic over a schedule is `flashcards/review`, and
the format is `markdown`.

## Writing a new rule

A rule is two things, and a rule missing either is not landed:

1. the walk over the tree, with a guard that fails when it reads nothing;
2. a negative control — a fixture written to be refused, asserting exactly what
   the rule catches and what it lets through.
