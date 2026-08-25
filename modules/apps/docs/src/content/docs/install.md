---
title: Install
description: Where to get numen for macOS, Windows and Linux, and where it keeps its own files.
---

Every build is on [numen.md](https://numen.md/#get). Pick the one for the machine in front of
you; the page marks the one it thinks you are on.

## macOS

Open the `.dmg` and drag numen into Applications.

## Windows

Run the installer. It brings the web view the window is drawn with, so nothing else has to be
installed first.

Windows warns about installers it has not seen before. The warning is about how many people have
run this file, not about what is in it: choose **More info**, then **Run anyway**.

There is also a plain `.zip` holding the program alone, for a machine where nothing is to be
installed.

## Linux

A `.deb` for Debian and Ubuntu, an `.rpm` for Fedora and openSUSE, and a `.tar.gz` holding the
program alone.

```sh
sudo apt install ./numen-linux-amd64.deb     # or
sudo dnf install ./numen-linux-amd64.rpm
```

## Updating

Install the new build over the old one. Your notes are your files and an install does not touch
them; what numen keeps of its own — the list of vaults, the settings, the search index — lives
outside the application and survives it.

## Where numen keeps its own files

| | |
| --- | --- |
| Linux | `~/.config/numen/` |
| macOS | `~/Library/Application Support/numen/` |
| Windows | `%AppData%\numen\` |

That folder holds [`numen.json`](/settings/), the list of vaults, the search index, and the
[`themes/`](/themes/) folder. Deleting it forgets which vaults you had and loses nothing you
wrote.

Each vault also carries a `.numen/` folder of its own, inside the folder of notes. It holds that
vault's identity — which is how numen recognises the same vault after the folder is moved or
renamed — and the text read out of any scanned [documents](/documents/).
