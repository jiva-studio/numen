# What the application says when it cannot start

An application starts by putting several things together: settings, a list of vaults, an index, an agent, the interface itself. Any of them can be missing or broken on somebody's machine. What the application cannot do, it says — where the person is, and with what it knows.

## Refusal is for what stops the work

Two states stop it:

- **settings that cannot be read** — a file that is not JSON, or one naming a size the setting it sits in does not take. A size named on the command line that the setting does not take stops the launch the same way;
- **an index that cannot be opened.**

Everything else the application starts with is something it can do without and name: a vault it does not have, an agent nobody can reach, a model that did not answer, a runtime it could not prepare, a theme whose name matches nothing, a size written under the old field name and out of range. Each is carried on without and said — in the window where the person is, or on standard error where the window is not up yet.

A settings file that is not there is not a refusal. It is written, holding exactly what this run is doing.

## A refusal is drawn in a window

The application opens one and puts in it what stopped it, the facts it holds about the state it found, and what to do about it. It writes one line to standard error as well, for a terminal and for a log, and leaves with a failing status.

The facts are the path of the index, the version of this build, the revision it was built from, and — where a schema is what stopped it — the number the index holds beside the number this build knows. See [ADR-0007](adr/0007-a-schema-change-is-a-numbered-migration.md) for what a build does about a schema it cannot account for.

The refusal window is a second interface, small and separate from the one the application serves. It has to keep working when the first one cannot be built, so it carries its own markup and its own styling, and reaches nothing shared.

An out-of-range `appearance.interface_scale` or `appearance.text_scale` is answered by this same window, naming the field, what was written and how far the setting goes. The number is left as it was written — see [Settings](settings.md).

## A refusal names no remedy that costs the person something

"Update numen" is a remedy. "Delete your index" is a bill, and it is not this application's to write. An index is never spent to recover from a state nobody diagnosed.

## A first run is given somewhere to write

An installation with no vault on its list is given one. A folder named `numen` is made where the system says this person keeps documents, it is given an identity, it goes on the list, and the window opens on it. Where it is is written to standard output.

It is theirs: ordinary files in an ordinary place, which they may move or replace with one of their own. A folder that is already a vault keeps the identity it has.
