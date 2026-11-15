# vm-runners

GitHub Actions self-hosted runners in Tart virtual machines on a Mac host. [runscaler](https://github.com/ysya/runscaler) registers a runner scale set, starts one fresh Tart clone of the image `numen-macos-26` for each job and deletes the clone after the job.

| Path | Contents |
| --- | --- |
| `images/macos/build.sh` | Builds the image `numen-macos-26` |
| `images/macos/provision.sh` | Runs inside the image build VM |
| `images/numen/flake.nix` | Tools installed with `nix profile install`: Node.js 22 and git |
| `runscaler/config.example.toml` | runscaler config with placeholders |

Apple allows two macOS VMs at a time per host, so `max-runners` must not be above 2.

## Build the image

On the Mac host, with Tart installed:

```sh
tools/vm-runners/images/macos/build.sh
```

The script clones `ghcr.io/cirruslabs/macos-tahoe-base:latest` (33 GB on the first pull) to `numen-macos-26-building`, boots it headless and provisions it with `tart exec`: Nix (Determinate installer), the profile from `images/numen/flake.nix`, Playwright Chromium and WebKit of the version in `package-lock.json`, and the newest GitHub Actions runner for osx-arm64 in `/Users/admin/actions-runner`, checked against the sha256 in its release notes. `actions-runner/.env` sets the job `PATH` so that Node.js 22 from the Nix profile comes before Homebrew's. Then the script syncs and halts the guest, waits until the VM is stopped and renames it to `numen-macos-26`. The old image is deleted only after that. On failure the `-building` VM is deleted.

`IMAGE_NAME`, `BASE_IMAGE`, `CPU`, `MEMORY`, `RUNNER_VERSION` and `RUNNER_SHA256` override the defaults.

## Install runscaler

1. Create a GitHub App for the organization with the permission Self-hosted runners (organization, read and write), install it on the organization and download its private key.
2. Put the key at `~/.config/runner/app.pem` with mode 600. It never goes into the repository.
3. Download the `darwin_arm64` tarball and `checksums.txt` of the latest release from <https://github.com/ysya/runscaler/releases>, check the tarball with `shasum -a 256 -c --ignore-missing checksums.txt`, unpack it and install the binary as `~/.local/bin/runner`.
4. Copy `runscaler/config.example.toml` to `~/.config/runner/config.toml`, set `url`, `runner-group` and the `github-app.*` values, and set the mode to 600.
5. Install and start the user service:

```sh
runner service install --scope user
runner status
```

## Refresh the image

When GitHub retires the runner version in the image, jobs stop being picked up. Run `build.sh` again. It takes the latest runner release and replaces the image. Running jobs keep their clones.
