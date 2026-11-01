# What the application reports when it cannot start

An application starts by putting several things together: settings, a list of vaults, an index, an agent, the interface itself. Any of them can be missing or broken on somebody's machine. What the application cannot do, it reports — in the place where the person is looking, with the information it has.

## Startup errors stop the work

Two states stop it:

- **settings that cannot be read** — a file that is not JSON, or one that sets a size the setting does not accept. A size named on the command line that the setting does not accept stops the launch the same way;
- **an index that cannot be opened.**

Everything else the application starts with is something it can run without and report: a vault it does not have, an agent nobody can reach, a model that did not respond, a runtime it could not prepare, a theme whose name matches nothing, a size written under the old field name and out of range. Each is skipped and reported, and the start continues — in the window where the person is, or on standard error where the window is not up yet.

A settings file that is not there is not an error. It is created, containing exactly the values this run uses.

## An error is displayed in a window

The application opens one and shows in it what stopped the start, what it found about the state, and what to do about it. It writes one line to standard error as well, for a terminal and for whatever collects a process that failed to start, and exits with a failing status.

The heading names what could not be opened: the index, or the person's own settings file, or nothing more particular than the application itself. No vault is named, because no vault caused the failure.

Four states are shown, each in the application's own words and each with a remedy:

- **an index that is not a database** — move the file aside, and numen creates a new one and fills it from the vaults;
- **an index path that is a folder** — point `-index` at a file, or move the folder out of the way;
- **an index folder nobody may write in** — the folder, and permission to write in it or a path somewhere the person can write;
- **settings that cannot be read** — the file, and either correcting it or moving it aside for a new one. A number that a size setting does not accept is reported with the field, the value written and the allowed range; one given on the command line is reported with the flag it was given under. The number is left as it was written — see [Settings](settings.md).

sqlite returns one message and one code for both a path that is a folder and a folder nobody may write in. They need two different fixes, so they are two states here, distinguished by what the path is on this machine.

Every state includes the version of this build and the revision it was built from, and the path of whichever file it is about.

The startup error window is a second interface, small and separate from the one the application serves. It has to keep working when the first one cannot be built, so it contains its own markup and its own styling, and uses nothing shared.

## An error names no remedy that costs the person something

"Update numen" is a remedy. "Delete your index" is a remedy that costs the person something, and the application does not suggest it. An index is never deleted to recover from a state that has not been diagnosed.

A file that is not a database is diagnosed. It is not an index, it contains nothing that a scan cannot recreate, and moving it aside is given as the remedy.

## A first run creates a vault

An installation with no vault on its list is given one. A folder named `numen` is created in the system's documents location for this person, it is given an identity, it is added to the list, and the window opens it. Where it is is written to standard output.

It contains ordinary files in an ordinary place, which the person may move or replace with another vault. A folder that is already a vault keeps the identity it has.
