# The plugin registry

The plugin catalog of numen.md (`/plugins`) is built from one JSON file, the registry, which `.github/workflows/plugin-registry.yml` collects every night and publishes at `https://dl.numen.md/plugins/registry/latest.json`, with a copy kept under the date (`plugins/registry/<YYYY-MM-DD>.json`). It lists the plugins of this repository and the plugins published from their own GitHub repositories, each with its manifest fields, its README, its screenshots, its changelog, the checks the registry runs and its test numbers.

## Plugins of this repository

Every folder of `plugins/` with a `manifest.json` is in the registry. Its entry takes:

- the manifest, read against the schema of `manifest.proto`: besides its id, name and permissions, the settings, commands, tab kinds, file types of its readers, text kinds, themes, slots, connection types and search functions it contributes; a slot entry is titled by its own title, else by the title of the plugin's command or handler it runs, else by the title of the shared command in `libs/shared/protocol/commands.json`;
- the downloads: the `dependencies` of the manifest, each with the size of its file for every platform, read from the Content-Length of a HEAD request, else from the Content-Range total of a GET of the first byte, at each source URL in turn until one answers; a file whose SHA-256 the previous registry has a size for is not asked again, and `--offline` asks none;
- `README.md`: the text before `<!-- more -->` is the introduction the catalog search and the page description use, and the whole README is the Overview of the plugin page;
- `screenshots/<name>-light.webp` and `screenshots/<name>-dark.webp`, one pair per screenshot;
- the changelog: the commits that touched the folder, grouped by the commit that changed the `version` of the manifest;
- the contributors: the people who wrote the commits that touched the folder, one per email address as `.mailmap` maps it, named as in their newest commit, with their commit counts, bots left out; online with a GitHub token, each also gets the login and avatar of their GitHub account from the commits API;
- the mutation score and the coverage of the nightly `mutation.yml` and `coverage.yml`, which publish them at `plugins/quality/mutation.json` and `plugins/quality/coverage.json`;
- the tests of the folder, counted from the source without running them: Go functions `Test…` and `Fuzz…` in `*_test.go`, TypeScript cases `it(` and `test(` in `*.test.ts` (an `.each` table counts once), and the stories exported from `*.stories.ts`, which run as browser tests; `node_modules`, `dist` and `testdata` are left out.

Every entry, internal or external, has an `icon`: the lucide name `icons` in `plugins/registry.json` gives for its id (`{ "icons": { "<id>": "<lucide name>" } }`), else the first icon its manifest declares on a tab kind, a command, a slot or a setting, else the icon of its category (`agents` bot, `connections` plug, `engines` cpu, `readers` book-open, `themes` palette, `tools` wrench, `views` layout-panel-left). A name that has no icon file in `@lucide/vue`, or an id of `icons` that is not in the registry, fails the collection; an external plugin whose manifest names an unknown icon keeps its previous entry, or is left out when it has none.

## Featured plugins

`featured` in `plugins/registry.json` lists the ids the catalog shows first, in order: the first three as cards with a screenshot, the first seven as the icons above the title. An id that is not in the registry fails the collection.

## Plugins of other repositories

`plugins/registry.json` lists the external plugins, each `{ "id", "repo", "author" }`: the plugin id, the GitHub repository as `owner/name` and the author's name. An id or a repository name equal to an id or a folder of `plugins/` is rejected, and only repositories of `jiva-studio` may use ids under `md.numen.`.

A repository of a plugin holds, at the tag of its latest GitHub release:

- `manifest.json`, whose `id` is the id listed in `plugins/registry.json`;
- `README.md`, with an introduction before `<!-- more -->`;
- `screenshots/<name>-light.webp` and `screenshots/<name>-dark.webp`;
- optionally `quality.json`, `{ "coverage": 82.5, "mutationScore": 71.0, "measuredAt": "<ISO time>", "tests": { "go": 40, "typescript": 120, "stories": 12 } }`, percentages and test counts the site shows as reported by the author;
- a LICENSE that GitHub recognizes.

The release carries one artifact per platform, named with the system (`darwin` or `macos`, `linux`, `windows`) and the architecture (`amd64`, `arm64`), for example `word-count-linux-amd64.tar.gz`. The notes of the releases are the plugin's changelog. A release is asked for with the ETag of the previous registry, and a release with the same tag and publication time keeps its previous entry and is not downloaded again.

## Checks

The registry runs these checks itself and shows them on the Quality tab of each plugin page, above the tests:

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
