---
title: Installation
description: Where to get numen for macOS, Windows and Linux, and where it keeps its own files.
---

Every build is on [numen.md](https://numen.md/#get). Pick the build for your machine; the page marks the one it detects for you.

## macOS

Open the `.dmg` and drag numen into Applications.

## Windows

Run the installer. It installs the web view the window uses, so nothing else has to be
installed first.

Windows warns about installers it has not seen before. The warning is about how many people have
run this file, not about what is in it: choose **More info**, then **Run anyway**.

There is also a plain `.zip` containing the program alone, for a machine where nothing is to be
installed.

## Linux

Use the package your system installs. If none of them fits, the flatpak runs on any distribution.

### Debian, Ubuntu, Fedora, openSUSE

```sh
sudo apt install ./numen-linux-amd64.deb     # or
sudo dnf install ./numen-linux-amd64.rpm
```

### Flatpak

One file for any machine that has flatpak on it, including everything it needs to display the window. Use it where the `.deb` or the `.rpm` fails to install.

```sh
flatpak install --user ./numen-linux-amd64.flatpak
flatpak run md.numen.Numen
```

The cards run in a window of their own: `flatpak run --command=numen-flashcards md.numen.Numen`.

A vault under `/media` or `/run/media` is accessible as it is; a vault anywhere else outside your home folder is not accessible until you allow it:

```sh
flatpak override --user --filesystem=/where/the/vault/is md.numen.Numen
```

### Snap

```sh
sudo snap install --dangerous ./numen-linux-amd64.snap
```

A snap can access your home folder and, once you allow it, a disk you plug in:

```sh
sudo snap connect numen:removable-media
```

### Arch

From the AUR:

```sh
paru -S numen-bin        # or: yay -S numen-bin
```

### Nix, NixOS

The flake is published next to the builds.

```sh
nix run tarball+https://dl.numen.md/latest/numen-nix.tar.gz
```

On NixOS, add it as an input and use its package:

```nix
{
  inputs.numen.url = "tarball+https://dl.numen.md/latest/numen-nix.tar.gz";

  # in the configuration:
  environment.systemPackages = [ inputs.numen.packages.x86_64-linux.numen ];
}
```

`latest/` is whatever release is newest, and your lock file pins the one you installed, and `nix flake update numen` updates to the newest. A specific release is available by its version, as `https://dl.numen.md/releases/<version>/numen-nix.tar.gz`.

### Anything else

A `.tar.gz` containing the program alone. Unpack it anywhere and run `numen`.

## Updating

When a newer build is published, a version badge appears in the title bar, after its title. It
shows the release date, the way to install on your system and the release notes. It comes from the
Update Notifier plugin, whose settings choose the channel (**Stable** or **Beta**), the address of
the release feed and the hours between two checks. A machine without a network shows nothing.

Install the new build over the old one. Your notes are your files and an install does not touch
them; what numen keeps of its own — the list of vaults, the settings, the search index — is stored
outside the application and is kept.

## Where numen keeps its own files

|                    |                                           |
| ------------------ | ----------------------------------------- |
| Linux              | `~/.config/numen/`                        |
| Linux, the flatpak | `~/.var/app/md.numen.Numen/config/numen/` |
| Linux, the snap    | `~/snap/numen/current/.config/numen/`     |
| macOS              | `~/Library/Application Support/numen/`    |
| Windows            | `%AppData%\numen\`                        |

A sandbox has its own folder, so a flatpak or a snap installed next to a `.deb` does not list the
vaults that the `.deb` has. Your notes are untouched by any of it: add the folder again and the same
vault comes back.

What is in it:

|               |                                                                                              |
| ------------- | -------------------------------------------------------------------------------------------- |
| `numen.json`  | [the settings](/settings/), yours to edit.                                                   |
| `vaults.json` | the vaults this installation knows and which one was open last. numen writes it; you do not. |
| `themes/`     | [your own themes](/themes/), one `.css` file each.                                           |

Deleting the whole folder removes the list of your vaults and loses nothing you wrote: the notes are your files, and adding the folders again brings the same vaults back.

The search index is not in there. It is `index.db` in the folder your system keeps application data in — `~/.local/share/numen/` on Linux, `~/Library/Application Support/numen/` on macOS, `%AppData%\numen\` on Windows. Everything in it was read from your files and can be read again. An index written by a newer Numen is never deleted: this Numen refuses to open it and asks you to update.

## What numen remembers

Which vault you had open, and the vaults on the list. That is all. When you open the window you get
the vault you left, with a new plex and a new agent conversation; the tabs you had are not
restored.

Each vault also contains a `.numen/` folder of its own, inside the folder of notes. It holds that
vault's identity — which is how numen recognises the same vault after the folder is moved or
renamed — and what each plugin creates from its files, in a folder named after the plugin: the text
read from scanned [documents](/documents/), the transcript of a recording.
