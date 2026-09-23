# What the application says when it cannot start

An application starts by putting several things together: settings, a list of vaults, an index, an agent, the interface itself. Any of them can be missing or broken on somebody's machine. What the application cannot do, it says — where the person is, and with what it knows.

## Refusal is for what stops the work

Two states stop it:

- **settings that cannot be read** — a file that is not JSON, or one naming a size the setting it sits in does not take. A size named on the command line that the setting does not take stops the launch the same way;
- **an index that cannot be opened.**

Everything else the application starts with is something it can do without and name: a vault it does not have, an agent nobody can reach, a model that did not answer, a runtime it could not prepare, a theme whose name matches nothing, a size written under the old field name and out of range. Each is carried on without and said — in the window where the person is, or on standard error where the window is not up yet.

A settings file that is not there is not a refusal. It is written, holding exactly what this run is doing.

## A refusal is drawn in a window

The application opens one and puts in it what stopped it, the facts it holds about the state it found, and what to do about it. It writes one line to standard error as well, for a terminal and for whatever collects a process that failed to start, and leaves with a failing status.

The heading names what could not be opened: the index, or the person's own settings file, or nothing more particular than the application itself. No vault is named, because no vault is what stopped it.

Four states are drawn, each in the application's own words and each carrying a remedy:

- **an index that is not a database** — move the file aside, and numen makes a new one and fills it from the vaults;
- **an index path that is a folder** — point `-index` at a file, or move the folder out of the way;
- **an index folder nobody may write in** — the folder, and permission to write in it or a path somewhere the person can write;
- **settings that cannot be read** — the file, and either putting it right or moving it aside for a new one. A number a size does not take names the field, what was written and how far the setting goes; one named on the command line is answered by the flag it was given under. The number is left as it was written — see [Settings](settings.md).

sqlite answers for a folder that is a path and a folder nobody may write in with one sentence and one code. They are two things to put right, so they are two states here, told apart by what the path on this machine is.

Every state carries the version of this build and the revision it was built from, and the path of whichever file it is about.

The refusal window is a second interface, small and separate from the one the application serves. It has to keep working when the first one cannot be built, so it carries its own markup and its own styling, and reaches nothing shared.

## A refusal names no remedy that costs the person something

"Update numen" is a remedy. "Delete your index" is a bill, and it is not this application's to write. An index is never spent to recover from a state nobody diagnosed.

A file that is not a database is diagnosed. It is not an index, it holds nothing a scan does not make again, and moving it aside is named as the way out.

## A first run is given somewhere to write

An installation with no vault on its list is given one. A folder named `numen` is made where the system says this person keeps documents, it is given an identity, it goes on the list, and the window opens on it. Where it is is written to standard output.

It is theirs: ordinary files in an ordinary place, which they may move or replace with one of their own. A folder that is already a vault keeps the identity it has.
