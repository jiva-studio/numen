# Publishing

Where the page, the manual and the builds are served from, and what has to
exist for the workflows to put them there.

## Three stores, three doors

Storage on bunny.net is never readable from the web: reading it needs the
zone's password, and the password is a secret. What serves a file to the world
is a **pull zone** standing in front of a storage zone. So a store is public
exactly to the extent that a pull zone points at it, and each of the three has
a store of its own — nothing in one can be reached through the other.

| | Storage zone | Pull zone | Served from |
| --- | --- | --- | --- |
| The page | `numen-site` | `numen-site` | `numen-site.b-cdn.net`, and `numen.md` |
| The manual | `numen-docs` | `numen-docs` | `numen-docs.b-cdn.net`, and `docs.numen.md` |
| The builds | `numen` | `numen-dl` | `numen-dl.b-cdn.net`, and `dl.numen.md` |

Every store is in Falkenstein, so the endpoint is `storage.bunnycdn.com`. Each
zone serves from every continent and holds what it was given for thirty days;
the builds' zone is on the volume tier, which is what large files that are
rarely fetched should be on.

The page and the manual go into the root of their stores. The builds go into three folders of theirs.

The manual is a folder of folders, each holding an `index.html`, and a pull
zone in front of storage answers `/install/` and `/install` alike with the file
inside. Nothing has to be configured for that.

## What is in the store

| Folder | What is in it |
| --- | --- |
| `latest/` | The newest release, under names that carry no version |
| `releases/<version>/` | That release under names that carry it, kept |
| `dev/` | The last build somebody asked for by hand |

Six files in each, and one name each in `latest/` and `dev/`:

| File | What it is |
| --- | --- |
| `numen-macos.dmg` | The window, universal, signed and notarized |
| `numen-windows-setup.exe` | The installer, with the webview bootstrapper |
| `numen-windows-amd64.zip` | The window on its own |
| `numen-linux-amd64.deb` | Debian and Ubuntu |
| `numen-linux-amd64.rpm` | Fedora and RHEL |
| `numen-linux-amd64.tar.gz` | The window on its own |

A name in `latest/` never carries a version, so a link written on the page once goes on working after every release, and each run writes over the same names. In `releases/` the same file is `numen-<version>-macos.dmg` and its neighbours, and what is written there is written once.

Beside the six, `latest.json` says the version, the build number, the channel, when it was released, and for each file its platform, architecture, kind, size, SHA-256, and the address of the copy under `releases/` that keeps this version. That last one is what a machine asking what is newest is sent to fetch.

A platform that was not built in a run leaves its files alone: the run writes what it made and nothing else.

## What a number means

A release is named by a git tag, `v0.4.0-alpha.1`, and that tag is the only place the version is written down. Beside it stands the build number: how many commits stand behind the one that was built, which rises with every commit and never repeats.

| Where | What it carries |
| --- | --- |
| The file names in `releases/` | The version |
| `numen --version` | `numen 0.4.0-alpha.1 (build 364)` |
| The corner of the welcome screen | The version |
| macOS `CFBundleShortVersionString` | The version |
| macOS `CFBundleVersion` | The build number, which is what macOS orders two builds of one version by |
| Windows `VIProductVersion` | `0.4.0.364` — four numbers, the last of them the build |
| The deb and the rpm | `0.4.0~alpha.1`, a tilde being where both managers sort a prerelease before the release it leads to |
| `latest.json` | The version and the build number, as two fields |

The version goes into the binary at the link, and a name the linker does not find is one it passes over in silence, so the Linux job asks the binary it just built what it calls itself and stops if the answer does not carry the version. It goes into the page as `VITE_NUMEN_VERSION` while the page is built, which is where the welcome screen reads it from.

## Where the page looks for them

`PUBLIC_NUMEN_DOWNLOADS_BASE`, read when the page is built and set to
`https://dl.numen.md`. Unset, it is `https://numen-dl.b-cdn.net`.

## What the workflows are told

Variables:

| Variable | Value |
| --- | --- |
| `BUNNY_STORAGE_ENDPOINT` | `storage.bunnycdn.com` |
| `BUNNY_STORAGE_ZONE` | `numen` |
| `BUNNY_PULL_ZONE_ID` | `6406732` |
| `BUNNY_SERVED_FROM` | `https://dl.numen.md` |
| `BUNNY_SITE_STORAGE_ZONE` | `numen-site` |
| `BUNNY_SITE_PULL_ZONE_ID` | `6406733` |
| `BUNNY_SITE_SERVED_FROM` | `https://numen.md` |
| `BUNNY_DOCS_STORAGE_ZONE` | `numen-docs` |
| `BUNNY_DOCS_PULL_ZONE_ID` | `6415145` |
| `BUNNY_DOCS_SERVED_FROM` | `https://docs.numen.md` |
| `PUBLIC_NUMEN_DOWNLOADS_BASE` | `https://dl.numen.md` |

Secrets: `BUNNY_STORAGE_KEY`, `BUNNY_SITE_STORAGE_KEY` and
`BUNNY_DOCS_STORAGE_KEY` are the three stores' own passwords; `BUNNY_API_KEY`
is the account key, which is what purging asks for. They are in 1Password
beside the account key they came from.

Each workflow names every one of these before it does anything, so a missing
one is reported by name rather than as a refusal further down.

## How it happens

**The page.** `landing.yml` builds it on every pull request. On the default
branch it also uploads what was built, takes down anything in the store that
this build did not write, purges the edge, and then fetches the page over the
public address to prove a stranger can read it.

**The manual.** `docs.yml` does the same for `modules/apps/docs`, and checks
one thing more before it builds: that the keyboard page still says what the
window does. It runs on a pull request touching the manual and on the three
files the keyboard page is written from.

**The builds.** `package.yml` runs on a tag beginning with `v` and builds every platform, and it runs when it is asked for by hand and builds what it was asked for. A tag is named by the tag and goes into `latest/` and `releases/`; a run asked for by hand is named by what it was asked for, goes into `dev/`, and publishes only when it was told to. Either way it renames what each platform produced to the names above, writes `latest.json`, uploads, purges, and fetches every file back over the public address, checking that what comes down is the size of what went up.

`latest/latest.json` is the one file asked for over and over by machines that already hold the rest, and the pull zone has to be told to hold it no longer than a minute. Everything under `releases/` is written once, and the zone can hold it for as long as it likes.

## The names on the web

`numen.md`, `www.numen.md`, `docs.numen.md` and `dl.numen.md` answer, each with
a certificate of its own, and each is what its `SERVED_FROM` variable names. A
name is put on the web in three steps: add the hostname to the pull zone, point
a CNAME at the zone's `b-cdn.net` address, and ask for the free certificate —
which is refused until the CNAME resolves.
