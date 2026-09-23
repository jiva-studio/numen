---
title: Installation
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

Take the package your system installs. Where none of them fits, the flatpak runs anywhere.

### Debian, Ubuntu, Fedora, openSUSE

```sh
sudo apt install ./numen-linux-amd64.deb     # or
sudo dnf install ./numen-linux-amd64.rpm
```

### Flatpak

One file for any machine that has flatpak on it, bringing along everything it needs to draw. This is the one to take where the `.deb` or the `.rpm` refuses to install.

```sh
flatpak install --user ./numen-linux-amd64.flatpak
flatpak run md.numen.Numen
```

The cards run in a window of their own: `flatpak run --command=numen-flashcards md.numen.Numen`.

A vault under `/media` or `/run/media` is reached as it is; a vault anywhere else outside your home folder is not, until you say so:

```sh
flatpak override --user --filesystem=/where/the/vault/is md.numen.Numen
```

### Snap

```sh
sudo snap install --dangerous ./numen-linux-amd64.snap
```

A snap reaches your home folder and, once you allow it, a disk you plug in:

```sh
sudo snap connect numen:removable-media
```

### Arch

From the AUR:

```sh
paru -S numen-bin        # or: yay -S numen-bin
```

### Nix, NixOS

The flake stands beside the builds.

```sh
nix run tarball+https://dl.numen.md/latest/numen-nix.tar.gz
```

On NixOS, name it as an input and take the package from it:

```nix
{
  inputs.numen.url = "tarball+https://dl.numen.md/latest/numen-nix.tar.gz";

  # in the configuration:
  environment.systemPackages = [ inputs.numen.packages.x86_64-linux.numen ];
}
```

`latest/` is whatever release is newest, and your lock file holds the one you took; `nix flake update numen` moves to the newest. One release and no other is named by its own version, as `https://dl.numen.md/releases/<version>/numen-nix.tar.gz`.

### Anything else

A `.tar.gz` holding the program alone. Unpack it wherever you keep such things and run `numen`.

## Updating

Install the new build over the old one. Your notes are your files and an install does not touch
them; what numen keeps of its own — the list of vaults, the settings, the search index — lives
outside the application and survives it.

## Where numen keeps its own files

| | |
| --- | --- |
| Linux | `~/.config/numen/` |
| Linux, the flatpak | `~/.var/app/md.numen.Numen/config/numen/` |
| Linux, the snap | `~/snap/numen/current/.config/numen/` |
| macOS | `~/Library/Application Support/numen/` |
| Windows | `%AppData%\numen\` |

A sandbox keeps its own folder, so a flatpak or a snap installed beside a `.deb` does not see the
vaults that one knows. Your notes are untouched by any of it: add the folder again and the same
vault comes back.

What is in it:

| | |
| --- | --- |
| `numen.json` | [the settings](/settings/), yours to edit. |
| `vaults.json` | the vaults this installation knows and which one was open last. numen writes it; you do not. |
| `themes/` | [your own themes](/themes/), one `.css` file each. |
| `agents.json` | where [an agent](/connect/) reaches the vault, and the token to present. It is there while the window is running. |
| `agents.token` | that token, kept so the line in your agent's configuration goes on working. |

Deleting the whole folder forgets which vaults you had and loses nothing you wrote: the notes
are your files, and adding the folders again brings the same vaults back.

The search index is not in there. It is `index.db` in the folder your system keeps caches in —
`~/.cache/numen/` on Linux — because that is what it is: everything in it was read out of your
files and can be read again.

## What numen remembers

Which vault you had open, and the vaults on the list — that is all. Open the window and you get
the vault you left, with a fresh plex and a fresh agent beside it; the tabs you had are not put
back.

Each vault also carries a `.numen/` folder of its own, inside the folder of notes. It holds that
vault's identity — which is how numen recognises the same vault after the folder is moved or
renamed — and the text read out of any scanned [documents](/documents/).
