# The core — architecture and invariants

The conventions every language in this repository shares are in `AGENTS.md` at
the root. This file defines specific rules for `libs/core/kernel`. Where a piece of code goes and how it is shaped is the skill [`nm-place-core`](../../../.agents/skills/nm-place-core/SKILL.md).

The architecture follows [`.agents/rules/architecture.md`](../../../../.agents/rules/architecture.md)
("Ports and adapters"). The following table summarizes architectural invariants
and the corresponding tests enforcing them.

## Layer boundaries and dependency direction

`container/layers_test.go` and `container/direction_test.go` enforce these boundaries.
Each test verifies adherence to structural rules.

| The rule                                                                      | The test                                           |
| ----------------------------------------------------------------------------- | -------------------------------------------------- |
| A package is on the layer its path states, and imports only from lower layers | `TestTheLayersAreWhatTheyAre`                      |
| No package that the core is compiled from imports an adapter                  | `TestNothingTheCoreIsCompiledFromReachesOutward`   |
| A use case imports only the use cases `useCaseEdges` declares for it          | `TestUseCaseImportsFollowDeclaredEdges`            |
| An adapter does not name the port it implements                               | `TestNoAdapterNamesThePortItSatisfies`             |
| The composition root returns no adapter type                                  | `TestTheRootPassesOutNoAdapterTheCompilerContains` |
| A package is imported under the name its layer gives it                       | `TestNoPackageIsImportedBelowItsLayer`             |
| No package declares an alias of a type another package of the core declares   | `TestTheTypeCheckedWorkspaceObeysItsRules/NoCoreTypeIsReexported` |
| An adapter that references a use case is driving, one with a type that implements a port is driven, and none is both | `TestEveryAdapterHasOneRoleAndADrivingOneReachesNoPort/AnAdapterIsDrivingOrDrivenNotBoth` |
| A driving adapter references no type, function, variable, constant or method declared under `port/`, aliases resolved | `TestEveryAdapterHasOneRoleAndADrivingOneReachesNoPort/ADrivingAdapterReferencesNoPort` |
| The composition root is the one package `container`; no package lies below it | `TestTheTypeCheckedWorkspaceObeysItsRules/TheCompositionRootIsOnePackage` |

## What the core may not do

| The rule                                                                 | The test                                    |
| ------------------------------------------------------------------------ | ------------------------------------------- |
| The core does not access the machine: no `os`, no filesystem, no network | `TestNothingOfTheCoreReachesTheMachine`     |
| Nothing reads the machine's clock — an instant is passed as an argument  | `TestNothingOfTheCoreReadsTheMachinesClock` |
| Nothing logs                                                             | `TestNothingOfTheCoreLogs`                  |

## Ports

A port is declared where it is consumed, and every method on one is called from
outside `port/`.

| The rule                                           | The test                                        |
| -------------------------------------------------- | ----------------------------------------------- |
| Every port is referenced somewhere else            | `TestPorts_EveryPortTypeIsReferenced`           |
| Every type a port declares is named somewhere else | `TestEveryTypePortDeclaresIsNamedSomewhereElse` |
| Every port method is called from outside `port/`   | `TestEveryPortMethodIsCalledSomewhereElse`      |
| Every adapter serving the schema is a driving one  | `TestEveryAdapterServingTheSchemaIsDriving`     |
| No function or method of a port calls a method of an interface value | `TestTheTypeCheckedWorkspaceObeysItsRules/APortDeclaresAndDoesNotOrchestrate` |
| An interface has at most the interfacebloat `max` of `.golangci-structure.yml` methods, embedded ones included, in every module that configuration covers | `TestTheTypeCheckedWorkspaceObeysItsRules/InterfaceSizeCountsTheWholeMethodSet` |

## Names in the core

| The rule                                                                                      | The test                                                                                   |
| --------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| An identity is named with the core's own term, in fields and in signatures                    | `TestTheCoresIdentitiesAreItsOwn`, `TestTheIdentitiesAPortAsksForAreTheCoresOwn`           |
| An instant is a `time.Time`                                                                   | `TestEveryInstantTheCoreCarriesIsATime`                                                    |
| A driving adapter calls no port outside its wiring files, and is covered by the depguard rule | `TestNoDrivingAdapterCallsAPortOutsideItsWiring`, `TestEveryDrivingAdapterIsOnTheLintRule` |
| The core starts no timer of the machine                                                       | `TestNothingOfTheCoreReadsTheMachinesClock`                                                |
| The core probes no value for another interface                                                | `TestNoCodeOfTheCoreProbesAValueForAnotherInterface`                                       |
| A port names no type from `internal/`                                                         | `TestNoPortNamesATypeFromInternal`                                                         |

No type is named by a gerund or a participle.

A concept is named with the standard term of its field.

## Constructors

A use case is created with everything it needs, or it is not created. The constructor
takes the collaborators, returns an error for what it cannot create, and stores
them in unexported fields — so no half-built use case exists for a method to check.

| The rule                                                                                 | The test                                     |
| ---------------------------------------------------------------------------------------- | -------------------------------------------- |
| A use case does not store a collaborator that its constructor takes in an exported field | `TestNoUseCaseCarriesACollaboratorInTheOpen` |

No use case exposes a collaborator.

A dependency that may genuinely be absent is not stored as nil in a field: the
composition root decides it explicitly — `ProofreadingConfig.Recognition`
returns a boolean flag (e.g. `isConfigured`), so a window that cannot proofread never receives a
proofreading dependency.

A policy is not a collaborator. How many pages a batch holds, where progress is
reported, what to do after a write — those are set on the created value and have
a default.

## Where a rule lives

A rule about one value is a method on that value. A rule over several is a
function in the package that defines the subject. `domain/` contains what is true of
a note, and the format rules are in `markdown`.

## Writing a new rule

A rule is two things, and a rule missing either is not merged:

1. the walk over the tree, with a guard that fails when it reads nothing;
2. a negative control — a fixture written to be rejected, asserting exactly what
   the rule catches and what it lets through.
