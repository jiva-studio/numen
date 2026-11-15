# vm-runners

GitHub Actions self-hosted runners in Tart virtual machines on a Mac host. [runscaler](https://github.com/ysya/runscaler) registers a runner scale set, starts one fresh Tart clone of the image `numen-macos-26` for each job and deletes the clone after the job.

| Path | Contents |
| --- | --- |
| `images/macos/build.sh` | Recipe for the image `numen-macos-26` |
| `images/macos/provision.sh` | Runs inside the image build VM |
| `images/numen/flake.nix` | Tools installed with `nix profile install`: Node.js 22 and git |
| `runscaler/config.example.toml` | runscaler config with placeholders |

Apple allows two macOS VMs at a time per host, so `max-runners` must not be above 2.

## The image

The image `numen-macos-26` on the host was made by hand from `ghcr.io/cirruslabs/macos-tahoe-base:latest` and contains: Nix with the profile from `images/numen/flake.nix` (Node.js 22 and git), Playwright Chromium and WebKit of the version in `package-lock.json`, the Claude Code CLI in `/Users/admin/.local/bin`, and the GitHub Actions runner for osx-arm64 in `/Users/admin/actions-runner` with a `.env` that sets the job `PATH` (Nix profile before Homebrew, so Node.js 22 wins over Homebrew's Node.js 24).

`images/macos/build.sh` and `provision.sh` are the written recipe for the same image, without Claude Code. They have not been run to the end: the one run on the host lost the `tart exec` connection while Nix was installing. Check them before relying on them.

When GitHub retires the runner version in the image, jobs stop being picked up. Update the runner in `/Users/admin/actions-runner` of the image (or rebuild it).

## Install runscaler

1. Create a GitHub App for the organization with the permission Self-hosted runners (organization, read and write), install it on the organization and download its private key.
2. Put the key at `~/.config/runner/app.pem` with mode 600. It never goes into the repository.
3. Download the `darwin_arm64` tarball and `checksums.txt` of the latest release from <https://github.com/ysya/runscaler/releases>, check the tarball with `shasum -a 256 -c --ignore-missing checksums.txt`, unpack it and install the binary as `~/.local/bin/runner`. Start it by that full path: another program called `runner` can come first in `PATH`.
4. Copy `runscaler/config.example.toml` to `~/.config/runner/config.toml`, set `url`, `runner-group` and the `github-app.*` values, and set the mode to 600.
5. Install and start the user service:

```sh
~/.local/bin/runner service install --scope user
~/.local/bin/runner status
```

The service is the LaunchAgent `io.github.ysya.runner`. Restart it with `launchctl kickstart -k gui/$(id -u)/io.github.ysya.runner`.
