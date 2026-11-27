# Plugin asset mirror

Plugin manifests download runtime assets (the ONNX Runtime archives and the model files of the engine plugins) from upstream hosts. This tool lists every such asset, builds a local copy of all of them, and checks a copy against the manifests. The copy is the fallback source of a manifest and the recovery set when an upstream host drops a release.

## Files

- `assets.mjs`: reads every `dependencies` array of every `plugins/*/manifest.json` and lists each unique asset: id, kind, version, platform, source URLs, `sha256`, archive member and member `sha256`, the plugins that consume it, and its mirror path.
- `inventory.json`: the committed inventory (no binaries). Plain data; it records what the manifests declare so a change of a manifest asset is visible in review.
- `archive.mjs`: finds the archive member the way the host does and hashes it (`.tar.gz`, `.tgz`, `.zip`).
- `mirror.mjs`, `run.mjs`: building and verifying a mirror directory, and the command line.
- `mirror.test.mjs`: tests on fixtures and a local server; no network.

## Commands

Run from the repository root with Node (no dependencies to install).

| Command                                                  | Network              | What it does                                                                                                                                                                                                                                                                                                             |
| -------------------------------------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `node tools/plugin-mirror/run.mjs list`                  | no                   | Prints each asset with its checksum, consumers and URLs.                                                                                                                                                                                                                                                                 |
| `node tools/plugin-mirror/run.mjs inventory`             | no                   | Fails when `inventory.json` differs from the manifests. This is the registry check `ts/plugin-mirror-inventory`.                                                                                                                                                                                                         |
| `node tools/plugin-mirror/run.mjs inventory --write`     | no                   | Rewrites `inventory.json` after a manifest asset changed.                                                                                                                                                                                                                                                                |
| `node tools/plugin-mirror/run.mjs fetch --mirror <dir>`  | yes, run by a person | Downloads every asset not yet in `<dir>`, trying its URLs in order, and keeps a file only when its size is recorded and its `sha256` (and member `sha256`) match the manifest. Assets no URL can supply are printed as `unavailable` and the exit code is 1. A rerun fetches only what is missing or fails verification. |
| `node tools/plugin-mirror/run.mjs verify --mirror <dir>` | no                   | Checks an existing mirror against the manifests: every asset present, size equal to `mirror.json`, `sha256` and member `sha256` recomputed. Files no manifest names are warned about.                                                                                                                                    |
| `node tools/plugin-mirror/run.mjs urls --base <url>`     | no                   | Prints the URL each asset has on a host that serves the mirror directory at `<url>`.                                                                                                                                                                                                                                     |

## Mirror layout

```
<mirror>/
  mirror.json                                   size, sha256 and the source URL each file was fetched from
  <id>/<version>/<os>-<arch>/<file name>        onnxruntime/1.23.0/linux-amd64/onnxruntime-linux-x64-1.23.0.tgz
```

A dependency declared for every platform uses `any` in place of `<os>-<arch>`. The file name is the last path segment of the first source URL. The layout is the URL path too: a host that serves `<mirror>` at `https://host/base/` serves an asset at `https://host/base/<path>`.

## Consuming the mirror

A dependency source lists `sourceUrls`: the host tries them in order and takes the first file whose checksum matches. The upstream URL stays first and the mirror URL is appended:

```json
"sourceUrls": [
  "https://github.com/microsoft/onnxruntime/releases/download/v1.23.0/onnxruntime-linux-x64-1.23.0.tgz",
  "https://mirror.example/base/onnxruntime/1.23.0/linux-amd64/onnxruntime-linux-x64-1.23.0.tgz"
]
```

The checksum in the manifest is the single source of truth, so a mirror that serves wrong bytes is rejected like any other source. The manifest fields are described in `docs/plugins/README.md`.

## Keeping it up to date

1. Change the asset in the manifest (new version, platform or URL).
2. Run `inventory --write` and commit `inventory.json` with the manifest. The check `ts/plugin-mirror-inventory` fails a branch where they differ.
3. A person with network access runs `fetch --mirror <dir>`, then `verify --mirror <dir>`, and uploads `<dir>` to the host.
4. Run `urls --base <url>` to get the fallback URLs and add them to `sourceUrls`.

Binaries stay out of Git: only `inventory.json` is committed.

## Open decisions

- Where the mirror is hosted (object storage bucket, release assets, internal host) and who holds its credentials. The tool only writes a directory; nothing uploads it.
- Whether a scheduled job runs `fetch` and `verify` against the hosted copy.
- An upstream artifact that is no longer available is reported by `fetch` as `unavailable`; its correct source is chosen by a person before the manifest URL changes.
