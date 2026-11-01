# Publishing

Where the page, the manual and the builds are served from, and what has to exist for the workflows to put them there.

## Three stores

Storage on bunny.net is never readable from the web: reading it needs the zone's password, and the password is a secret. What serves a file to the world is a **pull zone** placed in front of a storage zone. So a store is public exactly to the extent that a pull zone points at it, and each of the three has its own store — nothing in one store can be read through another store's pull zone.

|            | Storage zone | Pull zone    | Served from                                 |
| ---------- | ------------ | ------------ | ------------------------------------------- |
| The page   | `numen-site` | `numen-site` | `numen-site.b-cdn.net`, and `numen.md`      |
| The manual | `numen-docs` | `numen-docs` | `numen-docs.b-cdn.net`, and `docs.numen.md` |
| The builds | `numen`      | `numen-dl`   | `numen-dl.b-cdn.net`, and `dl.numen.md`     |

Every store is in Falkenstein, so the endpoint is `storage.bunnycdn.com`. Each zone serves from every continent and caches files for thirty days; the builds' zone is on the volume tier, which suits large files that are rarely fetched.

The page and the manual go into the root of their stores. The builds go into three folders of theirs.

The manual is a folder of folders, each holding an `index.html`, and a pull zone in front of storage returns the file inside for `/install/` and `/install` alike. Nothing has to be configured for that.

## What is in the store

| Folder                | What is in it                                       |
| --------------------- | --------------------------------------------------- |
| `latest/`             | The newest release, under names without a version   |
| `releases/<version>/` | That release under names containing the stamp, kept |
| `beta/`               | The newest beta, under names without a version      |
| `builds/<stamp>/`     | That beta under names containing the stamp, kept    |

Eight builds and the files that go with them, one name each in `latest/` and `beta/`:

| File                        | What it is                                                                        |
| --------------------------- | --------------------------------------------------------------------------------- |
| `numen-macos.dmg`           | The window, universal, signed and notarized                                       |
| `numen-windows-setup.exe`   | The installer, with the webview bootstrapper                                      |
| `numen-windows-amd64.zip`   | The window on its own                                                             |
| `numen-linux-amd64.deb`     | Debian and Ubuntu                                                                 |
| `numen-linux-amd64.rpm`     | Fedora and RHEL                                                                   |
| `numen-linux-amd64.flatpak` | Any distribution with flatpak installed; the browser comes from the GNOME runtime |
| `numen-linux-amd64.snap`    | Ubuntu; the browser is bundled inside                                             |
| `numen-linux-amd64.tar.gz`  | The two windows and the icons a desktop shows for them                            |
| `PKGBUILD`                  | The recipe an Arch machine uses to build a package from the archive               |
| `numen-nix.tar.gz`          | The flake a Nix machine reads; it refers to the archive above                     |
| `notes.md`                  | What this release contains that the last one did not                              |

The `PKGBUILD` and the flake are recipes: they are read rather than fetched by a machine, so they are stored under their own names and are not in the manifest. Each is written by the run that made the build it refers to, and contains that build's address and hash.

The repository is closed, which is why the flake is distributed as a file at all — `nix run tarball+https://dl.numen.md/latest/numen-nix.tar.gz` is what an outside user runs instead of using a git address. It holds `flake.nix`, `tools/build-nix/numen.nix`, the desktop entries and the icons, a `tools/build-nix/release.json` written by the run, and the lock: a flake downloaded as a file cannot create its own lock file.

What an Arch user installs is the AUR package `numen-bin`. Pushing the `PKGBUILD` to the AUR is done manually by a person; the AUR repository contains this file and nothing else from this project.

A name in `latest/` or `beta/` never contains a version, so a link written on the page keeps working after every release, and each run overwrites the same names. Every file is uploaded a second time under a name containing the stamp — `numen-2026.9.1-beta.884-884-283d1d5-macos.dmg` — and what is written under that name is written once and kept.

Beside the eight, `latest.json` records the version, the build number, the commit, the stamp, the channel, when it was released, and for each file its platform, architecture, kind, size, SHA-256, and the address of the stamped copy. A machine checking for the newest version fetches that address, and the page points its links to it, so a file taken from the page shows in its own name which build it is.

The files of a platform that was not built in a run are not changed: the run writes what it made and nothing else.

## What a number means

A release is named by a git tag, `v0.4.0-alpha.1`, and that tag is the only place the version is recorded. The build number is the count of commits up to the one that was built; it rises with every commit and never repeats. The third part is the commit itself, seven characters of it, which identifies the source and nothing else.

The three written with dashes between them are the stamp, `0.4.0-alpha.1-364-283d1d5`, and every file a run uploads contains it in its name.

| Where                              | What it contains                                                                       |
| ---------------------------------- | -------------------------------------------------------------------------------------- |
| The stamped file names             | The version, the build number and the commit                                           |
| `numen --version`                  | `numen 0.4.0-alpha.1 (build 364, commit 283d1d5)`                                      |
| The corner of the welcome screen   | The version                                                                            |
| macOS `CFBundleShortVersionString` | The version                                                                            |
| macOS `CFBundleVersion`            | The build number, which macOS uses to order two builds of one version                  |
| Windows `VIProductVersion`         | `0.4.0.364` — four numbers, the last of them the build                                 |
| The deb and the rpm                | `0.4.0~alpha.1`, because both package managers sort a tilde version before the release |
| The snap                           | The version as it is written                                                           |
| The `PKGBUILD`                     | `0.4.0_alpha.1`, a pkgver without a dash                                               |
| `latest.json`                      | The version, the build number, the commit and the stamp, as four fields                |

The version, the build number and the commit are set in the binary at link time, and the linker silently ignores a variable name it does not find, so the Linux job runs the binary it just built to print its version and stops if the output does not contain all three. The version is set in the page as `VITE_NUMEN_VERSION` while the page is built, and the welcome screen reads it from there.

## Where the page reads the downloads from

`PUBLIC_NUMEN_DOWNLOADS_BASE`, read when the page is built and set to `https://dl.numen.md`. Unset, it is `https://numen-dl.b-cdn.net`.

## What the workflows are given

Variables:

| Variable                      | Value                   |
| ----------------------------- | ----------------------- |
| `BUNNY_STORAGE_ENDPOINT`      | `storage.bunnycdn.com`  |
| `BUNNY_STORAGE_ZONE`          | `numen`                 |
| `BUNNY_PULL_ZONE_ID`          | `6406732`               |
| `BUNNY_SERVED_FROM`           | `https://dl.numen.md`   |
| `BUNNY_SITE_STORAGE_ZONE`     | `numen-site`            |
| `BUNNY_SITE_PULL_ZONE_ID`     | `6406733`               |
| `BUNNY_SITE_SERVED_FROM`      | `https://numen.md`      |
| `BUNNY_DOCS_STORAGE_ZONE`     | `numen-docs`            |
| `BUNNY_DOCS_PULL_ZONE_ID`     | `6415145`               |
| `BUNNY_DOCS_SERVED_FROM`      | `https://docs.numen.md` |
| `PUBLIC_NUMEN_DOWNLOADS_BASE` | `https://dl.numen.md`   |

Secrets: `BUNNY_STORAGE_KEY`, `BUNNY_SITE_STORAGE_KEY` and `BUNNY_DOCS_STORAGE_KEY` are the three stores' own passwords; `BUNNY_API_KEY` is the account key, which purging requires. They are stored in 1Password next to the account key they came from.

Each workflow checks every one of these before it does anything, so a missing one is reported by name before downstream operations fail.

## How the workflows run

**The page.** `site.yml` builds it on every pull request. On the default branch it also uploads what was built, deletes anything in the store that this build did not write, purges the edge, and then fetches the page over the public address to check that an anonymous visitor can read it.

**The manual.** `docs.yml` does the same for `apps/docs`, and checks one thing more before it builds: that the five generated pages still match what the application does. It runs on a pull request touching the manual and on the twelve files those pages are generated from.

**The builds.** `release.yml` is started manually and builds one channel. A stable release is named from the month it is made in and the number of stable releases already made that month — `2026.9.0`, then `2026.9.1` — and goes into `latest/` and `releases/`, which is where the page reads from. A beta is named from the stable release it precedes and the number of commits before it — `2026.9.1-beta.884` — and goes into `beta/` and `builds/`, which the page does not link to. Either way it renames what each platform produced to the names above, writes `latest.json`, uploads, purges, and fetches every file back over the public address, checking that each downloaded file is identical to the uploaded one.

The flatpak and the snap are made from what the Linux job built, each in a job of its own, and the Linux checkbox selects all three. The flatpak is installed from its own bundle before the run continues, and its version is queried, which shows that the GNOME runtime contains the browser the window is linked against.

The changes made since the last stable release are computed before anything is uploaded, and are uploaded next to the builds as `notes.md`, which is the address `latest.json` contains. The repository is closed, so that copy is the one a reader of the page can open.

Then it tags the commit it built and makes a release on GitHub with the notes, the canonical and stamped builds, `checksums.txt`, and `latest.json`. A beta's release is marked as a prerelease. The tag is made once the builds are up, so no tag points to a failed build.

`latest/latest.json` is the one file requested repeatedly by installed copies — among them the editor, whose Update Notifier plugin (`md.numen.tools.updater`) requests it every few hours from its own backend, or requests `beta/latest.json` on the beta channel, or the address set in its settings instead — and the pull zone has to be set to cache it for no longer than a minute — today it is set to thirty days. Everything under `releases/` and `builds/` is written once, and the zone can cache it for any length of time.

The page is on one host and the builds on another, so the builds' zone has to send `Access-Control-Allow-Origin` for the page to read the manifest at all. Until it does, the page keeps the links it was written with, which point to the newest build under names without a version.

## How long a copy is cached

Two settings on every pull zone, and they are different settings.

**The edge caches files for thirty days**, and each deploy purges the zone it wrote to, so a visitor receives what was last published.

**A browser caches for sixty seconds.** A page refers to the files it was built with, and those names contain a hash of the file content, so a browser that caches a page for longer refers to files that no longer exist: the page loads but its images do not. Sixty seconds is `CacheControlPublicMaxAgeOverride` on the zone, and it is the setting to look at when a change is published and somebody still sees the old version.

## The names on the web

`numen.md`, `www.numen.md`, `docs.numen.md` and `dl.numen.md` respond, each with its own certificate, and each is the value of its `SERVED_FROM` variable. A name is put on the web in three steps: add the hostname to the pull zone, point a CNAME at the zone's `b-cdn.net` address, and request the free certificate — which fails until the CNAME resolves.
