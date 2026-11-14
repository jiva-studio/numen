# Calls no check decides

## Is this rule in the domain or in a use case

- A rule that can be decided from values the caller already holds (a connection has two distinct notes, a role has an inverse, a name folds to a filename) is the domain's. It is a method on the value, or a function in the package that defines the subject. `libs/note/domain` holds what is true of a note; `markdown` holds the rules of the file's format.
- A rule that needs stored data is split: the deciding part is a domain function that takes the data as a parameter (a value or a function that answers a question), and the use case reads the data through a port and passes it.
- A step that orders several ports, handles their failures or reports progress is a use case. Test: if the step names a port, it is a use case.
- A choice among sources (stored value, then hint, then default) is a use case, because the sources are ports.
- A value that two use cases both need to validate is validated in the domain once.

## Is this one port or two

- Ask what each caller asks. Two groups of methods that no single use case calls together are two ports.
- Reading and writing of one aggregate are separate ports; a use case that reads takes the reader only.
- A port with more than three methods is cut along the callers before it is accepted. A long method list that a plan or an older note justifies is still cut along the callers.
- A new method is added to an existing port only when the use case that needs it already holds that port and asks the same question; otherwise it starts a new small port.

## Where a shared type lives

- A fact about notes, vaults, links, cards or text: `libs/note/domain`.
- The vocabulary of a port that no domain type holds: the port package that owns it, and only use cases and adapters of that port name it.
- The request and result of one scenario: its use case package.
- A type that appears in a driving adapter's signature but is declared under `port/` is moved: it is a result of a scenario and lives in the use case package, or it is a domain fact and lives in the domain.
- A name is declared once. A package does not re-declare another package's type under its own name to shorten an import; the importer writes the original package name.

## Use case or adapter

- Computation over data the core already holds (cutting text into chunks, ranking, deciding what is stale, merging results) is use case or domain work, or a pure helper in `internal/` that a use case calls.
- Translating one representation to another at a boundary (SQL rows to domain values, protocol messages to use case arguments, a plugin's JSON to a port type) is the adapter's work.
- A detail of a technology (a pragma, a lock file, a platform trash location, a process flag) is the adapter's and does not appear in a port's types.
- When in doubt, ask whether the code would change if the storage or the transport changed. If it would not, it is not an adapter.

## Settings

- The meaning and limits of a setting (a scale between two bounds, a model name, a floor above zero) are values with their checks in `libs/note/domain` or in the settings use cases.
- Reading, writing, watching and applying settings are use cases over the settings ports (`settingsport`), which hold the file's content as `Setting` values.
- The file's layout, locking, merge with the defaults and the declared schema are the settings adapter's.
- The consumer of a setting is handed the value, or a function that returns the current value, by `container`. No consumer opens the file, and `container` computes no setting.

## Time and identifiers

- The use case receives `machineport.Clock` and calls `Now()` once per scenario, then passes the instant on. Randomness needs no port in Go. A timer whose firing changes what a caller or a test observes (a wait, a quiet period, a timeout) takes `After` and `Wait` from the clock port (`machineport.Timer` in the core), so a test moves the time and the timer fires. A function in the domain takes the instant as an argument.
- A use case that needs a new identifier receives a port that returns one; the adapter builds the identifier with `ulid`. The domain receives identifiers, never creates them.

## What an application may keep

- Options and flags, the paths of the platform, the toolkit's window, menu and dialog code, the build of the `container.Config`, and the choice of which window or plugin target the binary serves.
- A small piece that is specific to one application, such as the plugin target `desktop.flashcards` that the flashcards window serves, stays in that application.
- Anything that decides what happens to a note, a vault or a card is a use case; the application calls it through `container`.
- A toolkit or operating system facility the core needs (a folder dialog, a keychain) is a driven adapter that implements a port; it lives in its own package under `apps/desktop/internal` and is bound in the application's wiring.
