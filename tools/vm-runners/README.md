# vm-runners

GitHub Actions self-hosted runners in Tart virtual machines on a Mac host. [GARM](https://github.com/cloudbase/garm) manages the runners through a runner scale set. GARM starts each runner VM through `garm-provider-tart`, an external provider. Each runner VM is a copy-on-write clone of a golden image and runs one job.

| Path | Contents |
| --- | --- |
| `garm-provider-tart/` | GARM external provider (Go, interface v0.1.0) that clones, starts, stops and deletes Tart VMs |
| `images/macos/` | Build script for the macOS golden image `numen-macos-26` and the bootstrap LaunchAgent baked into it |
| `images/numen/flake.nix` | Tools installed into the image with `nix profile install`: Node.js 22 and git |

## How a runner starts

1. GARM calls the provider with `GARM_COMMAND=CreateInstance` and the bootstrap parameters on stdin.
2. The provider writes `bootstrap.json` and `bootstrap.sh` into `<state_dir>/instances/<name>/shared`, clones the golden VM named by the pool image to a VM named after the instance, and starts `tart run --no-graphics --no-audio --dir=garm:<shared dir> <name>` in the background.
3. The guest logs in as `admin` automatically. The LaunchAgent `md.numen.runner-bootstrap` waits for `/Volumes/My Shared Files/garm/bootstrap.sh` and runs it once. Output goes to `bootstrap.log` in the shared directory, the exit code to `bootstrap.exit`.
4. `bootstrap.sh` downloads `<metadata-url>/install-script/` with the instance token and runs it. GARM renders that script from the runner install template of the pool or scale set.

The guest reaches the host at the Tart gateway address `192.168.64.1`. GARM's `metadata_url` and `callback_url` must use that address.

The provider keeps one directory per instance under `state_dir/instances/`: `instance.json` (controller ID, pool ID, image), `tart-run.pid`, `tart-run.log` and the shared directory. It only touches VMs that have such a directory, and only those of the calling controller.

## Build the macOS image

On the Mac host, with Tart and Nix installed:

```sh
tools/vm-runners/images/macos/build.sh
```

The script clones `ghcr.io/cirruslabs/macos-tahoe-base:latest` (about 27 GB on the first pull) to `numen-macos-26-building`, boots it, and provisions it over SSH with the image's default `admin`/`admin` login: Nix (Determinate installer), the tools from `images/numen/flake.nix`, Playwright Chromium and WebKit of the version in `package-lock.json`, and the bootstrap LaunchAgent. It then shuts the VM down and renames it to `numen-macos-26`, replacing an older image of that name. `IMAGE_NAME`, `BASE_IMAGE`, `CPU` and `MEMORY` override the defaults.

## Install the provider

```sh
cd tools/vm-runners/garm-provider-tart
GOWORK=off go build -o ~/garm/garm-provider-tart .
```

Provider config, for example `~/garm/garm-provider-tart.toml`:

```toml
tart = "/opt/homebrew/bin/tart"
state_dir = "/Users/runner/garm/tart"
cpu = 4
memory = 8192

[images]
numen-macos-26 = "numen-macos-26"
```

`images` maps the image name of a GARM pool or scale set to a local Tart VM.

Provider entry in the GARM config:

```toml
[[provider]]
name = "tart"
provider_type = "external"
description = "Tart VMs on this Mac"
  [provider.external]
  provider_executable = "/Users/runner/garm/garm-provider-tart"
  config_file = "/Users/runner/garm/garm-provider-tart.toml"
  interface_version = "v0.1.0"
```

GARM must run as the user who owns the Tart VMs, in a session that may use Virtualization.framework.

## Run the provider by hand

```sh
export GARM_PROVIDER_CONFIG_FILE=~/garm/garm-provider-tart.toml GARM_CONTROLLER_ID=test-controller GARM_POOL_ID=test-pool
GARM_COMMAND=CreateInstance ./garm-provider-tart < bootstrap.json
GARM_COMMAND=ListInstances ./garm-provider-tart
GARM_COMMAND=DeleteInstance GARM_INSTANCE_ID=<name> ./garm-provider-tart
```

`bootstrap.json` needs `name`, `image`, `pool_id`, `os_type`, `arch`, `metadata-url` and `instance-token`.
