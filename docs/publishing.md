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

The page and the manual go into the root of their stores. The builds go into
`latest/`.

The manual is a folder of folders, each holding an `index.html`, and a pull
zone in front of storage answers `/install/` and `/install` alike with the file
inside. Nothing has to be configured for that.

## What is in `latest/`

Six files, and one name each. The name never carries a version, so a link
written on the page once goes on working after every release, and the store
holds the newest build and nothing older — each run writes over the same names.

| File | What it is |
| --- | --- |
| `numen-macos.dmg` | The window, universal, signed and notarized |
| `numen-windows-setup.exe` | The installer, with the webview bootstrapper |
| `numen-windows-amd64.zip` | The window on its own |
| `numen-linux-amd64.deb` | Debian and Ubuntu |
| `numen-linux-amd64.rpm` | Fedora and RHEL |
| `numen-linux-amd64.tar.gz` | The window on its own |

Beside them, `latest.json` says the version, when it was built, and the size
and SHA-256 of each file.

A platform that was not built in a run leaves its files alone: the run writes
what it made and nothing else.

## Where the page looks for them

`PUBLIC_NUMEN_DOWNLOADS_BASE`, read when the page is built. Unset, it is
`https://numen-dl.b-cdn.net`; once `dl.numen.md` answers, it is that.

## What the workflows are told

Variables:

| Variable | Value |
| --- | --- |
| `BUNNY_STORAGE_ENDPOINT` | `storage.bunnycdn.com` |
| `BUNNY_STORAGE_ZONE` | `numen` |
| `BUNNY_PULL_ZONE_ID` | `6406732` |
| `BUNNY_SERVED_FROM` | `https://numen-dl.b-cdn.net` |
| `BUNNY_SITE_STORAGE_ZONE` | `numen-site` |
| `BUNNY_SITE_PULL_ZONE_ID` | `6406733` |
| `BUNNY_SITE_SERVED_FROM` | `https://numen-site.b-cdn.net` |
| `BUNNY_DOCS_STORAGE_ZONE` | `numen-docs` |
| `BUNNY_DOCS_PULL_ZONE_ID` | `6415145` |
| `BUNNY_DOCS_SERVED_FROM` | `https://numen-docs.b-cdn.net` |

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

**The builds.** `package.yml` is asked for by hand, and publishes only when it
is asked to. It renames what each platform produced to the names above, writes
`latest.json`, uploads, purges, and fetches every file back over the public
address, checking that what comes down is the size of what went up.

## The names on the web

`numen.md`, `www.numen.md` and `dl.numen.md` answer, each with a certificate of
its own. A name is put on the web in three steps: add the hostname to the pull
zone, point a CNAME at the zone's `b-cdn.net` address, and ask for the free
certificate — which is refused until the CNAME resolves.

`docs.numen.md` is attached to its pull zone and waits on the record
`docs → numen-docs.b-cdn.net`. Until it resolves, the manual is served from
`numen-docs.b-cdn.net`, which is what `BUNNY_DOCS_SERVED_FROM` names.
