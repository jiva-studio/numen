# The plugin registry

The plugin catalog of numen.md (`/plugins`) is built from one JSON file, the registry, which `.github/workflows/plugin-registry.yml` collects every night and publishes at `https://dl.numen.md/plugins/registry/latest.json`, with a copy kept under the date (`plugins/registry/<YYYY-MM-DD>.json`). It lists the plugins of this repository and the plugins published from their own GitHub repositories, each with its manifest fields, its README, its screenshots, its changelog, the checks the registry runs and its test numbers.

## Plugins of this repository

Every folder of `plugins/` with a `manifest.json` is in the registry. Its entry takes:

- the manifest, read against the schema of `manifest.proto`;
- `README.md`: the text before `<!-- more -->` is the introduction the catalog card shows, and the whole README is the Overview of the plugin page;
- `screenshots/<name>-light.webp` and `screenshots/<name>-dark.webp`, one pair per screenshot;
- the changelog: the commits that touched the folder, grouped by the commit that changed the `version` of the manifest;
- the mutation score and the coverage of the nightly `mutation.yml` and `coverage.yml`, which publish them at `plugins/quality/mutation.json` and `plugins/quality/coverage.json`.

## Plugins of other repositories

`plugins/registry.json` lists the external plugins, each `{ "id", "repo", "author" }`: the plugin id, the GitHub repository as `owner/name` and the author's name. An id or a repository name equal to an id or a folder of `plugins/` is rejected, and only repositories of `jiva-studio` may use ids under `md.numen.`.

A repository of a plugin holds, at the tag of its latest GitHub release:

- `manifest.json`, whose `id` is the id listed in `plugins/registry.json`;
- `README.md`, with an introduction before `<!-- more -->`;
- `screenshots/<name>-light.webp` and `screenshots/<name>-dark.webp`;
- optionally `quality.json`, `{ "coverage": 82.5, "mutationScore": 71.0, "measuredAt": "<ISO time>" }`, percentages the site shows as reported by the author;
- a LICENSE that GitHub recognizes.

The release carries one artifact per platform, named with the system (`darwin` or `macos`, `linux`, `windows`) and the architecture (`amd64`, `arm64`), for example `word-count-linux-amd64.tar.gz`. The notes of the releases are the plugin's changelog. A release is asked for with the ETag of the previous registry, and a release with the same tag and publication time keeps its previous entry and is not downloaded again.

## Checks

The registry runs these checks itself and shows them on the Scorecard of each plugin:

| Check | Internal | External |
| --- | --- | --- |
| `manifest`: the manifest matches the schema | yes | yes |
| `license` | the repository's | the repository's, from GitHub |
| `readme`: README with an introduction | yes | yes |
| `screenshots`: at least one, in light and dark | yes | yes |
| `permissions`: those the current version added | yes | yes |
| `updated`: the last commit or release | yes | yes |
| `artifacts`: a SHA-256 for each artifact | | yes |
| `attestation`: `gh attestation verify` of each artifact | | where gh is installed |
| `vulnerabilities`: osv-scanner over the source at the tag | | where osv-scanner is installed |

## Building it locally

```
node tools/plugin-registry/collect.mjs --out .registry --offline
NUMEN_PLUGIN_REGISTRY_FILE=$PWD/.registry/latest.json npm run build --prefix apps/landing
```

`--quality <dir>` reads `mutation.json` and `coverage.json` from a folder; `tools/plugin-registry/quality-from-run.mjs` writes that folder from the artifacts of a `mutation.yml` run and a `coverage.yml` run. Without NUMEN_PLUGIN_REGISTRY_FILE the site reads the published registry, and a registry it cannot read fails the build.
