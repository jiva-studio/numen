# The plugin registry

The plugin catalog of numen.md (`/plugins`) is built from one JSON file, the registry, which `.github/workflows/plugin-registry.yml` collects every night and publishes at `https://dl.numen.md/plugins/registry/latest.json`, with a copy kept under the date (`plugins/registry/<YYYY-MM-DD>.json`). It lists the plugins of this repository and the plugins published from their own GitHub repositories, each with its manifest fields, its README, its screenshots, its changelog, its third-party dependencies and their known vulnerabilities, the checks the registry runs and its test numbers.

## Plugins of this repository

Every folder of `plugins/` with a `manifest.json` is in the registry. Its entry takes:

- the manifest, read against the schema of `manifest.proto`: besides its id, name and permissions, the settings, commands, tab kinds, file types of its readers, text kinds, themes, slots, connection types and search functions it contributes; a slot entry is titled by its own title, else by the title of the plugin's command or handler it runs, else by the title of the shared command in `libs/shared/protocol/commands.json`;
- the downloads: the `dependencies` of the manifest, each with the size of its file for every platform, read from the Content-Length of a HEAD request, else from the Content-Range total of a GET of the first byte, at each source URL in turn until one answers; a file whose SHA-256 the previous registry has a size for is not asked again, and `--offline` asks none;
- `README.md`: the text before `<!-- more -->` is the introduction the catalog search and the page description use, and the whole README is the Overview of the plugin page;
- `screenshots/<name>-light.webp` and `screenshots/<name>-dark.webp`, one pair per screenshot;
- the changelog: the commits that touched the folder, grouped by the commit that changed the `version` of the manifest;
- the contributors: the people who wrote the commits that touched the folder, one per email address as `.mailmap` maps it, named as in their newest commit, with their commit counts, bots left out; online with a GitHub token, each also gets the login and avatar of their GitHub account from the commits API;
- the mutation score and the coverage of the nightly `mutation.yml` and `coverage.yml`, which publish them at `plugins/quality/mutation.json` and `plugins/quality/coverage.json`;
- the tests of the folder, counted from the source without running them: Go functions `Test…` and `Fuzz…` in `*_test.go`, TypeScript cases `it(` and `test(` in `*.test.ts` (an `.each` table counts once), and the stories exported from `*.stories.ts`, which run as browser tests; `node_modules`, `dist` and `testdata` are left out;
- the dependencies and their vulnerabilities, below.

## Dependencies

The Quality tab lists the third-party dependencies of a plugin, each with its version and SPDX license:

- the Go modules the `require` lines of its `go.mod` name without `// indirect`; the indirect ones are counted, not listed. The license is read from the license files (`LICENSE*`, `LICENCE*`, `COPYING*`, `UNLICENSE*`) at the root of the module, which `go mod download -json` puts in the module cache, by `tools/plugin-registry/licenses`, a Go program that classifies them with `github.com/google/licensecheck` and keeps the licenses that cover at least 75 percent of a file. `--offline` sets GOPROXY=off, so only modules already in the cache have a license; where Go is not installed none has;
- the npm packages of the `dependencies` of its `package.json`, not `devDependencies` or `peerDependencies`, at the version and with the `license` the root `package-lock.json` resolves them to; the packages those install, followed through `dependencies` and `optionalDependencies` as Node resolves them, are counted as indirect.

The modules and packages of this repository, `github.com/jiva-studio/numen/...` and `@numen/*`, are the Numen SDK and are not listed.

The vulnerabilities are those osv-scanner (`osv-scanner scan source --lockfile <file> --format json`) reports for the `go.mod` of the plugin, all its modules included, and those it reports in the root `package-lock.json` for the npm packages the plugin installs, one per group of aliases, each linked to its page on osv.dev. The Go standard library and the toolchain, which the go line of `go.mod` names, belong to the Go release and are left out.

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

The registry runs these checks itself and shows them on the Quality tab of each plugin page, above the vulnerabilities, the dependencies and the tests. The permissions are on the Contributions tab and the dates of the changes on the Changelog tab, not here.

| Check | Internal | External |
| --- | --- | --- |
| `manifest`: the manifest matches the schema | yes | yes |
| `license` | the repository's | the repository's, from GitHub |
| `readme`: README with an introduction | yes | yes |
| `screenshots`: at least one, in light and dark | yes | yes |
| `permissions`: a warning when the current version added permissions | yes | yes |
| `artifacts`: a SHA-256 for each artifact | | yes |
| `attestation`: `gh attestation verify` of each artifact | | where gh is installed |
| `tests`: passed, or a warning when no test was counted; the counts are in the tests section | yes | from `quality.json` |
| `mutation`: passed when every measured mutation score is at least 80 percent, else a warning; not run when none is measured; the scores are in the tests section | yes | from `quality.json` |
| `stories`: every Vue component is shown by a story | yes | in the source at the tag |
| `manual`: a page of the manual names the plugin | yes | |
| `network`: whether the plugin makes network requests (information, never a failure) | yes | in the source at the tag |
| `vulnerabilities`: no known vulnerability in the dependencies | online, where osv-scanner is installed | online, where osv-scanner and git are installed |

`stories`: a Vue component is a `.vue` file under `frontend/`, and it has a story when a `*.stories.ts` of the plugin imports that file by a relative path. Not run for a plugin without a `frontend/` folder or without components.

`manual`: a page of `apps/docs/src/content/docs` (`.md` and `.mdx`) names the plugin when its text has the manifest `name` or the `id`, as a whole word and in the same case. The blocks `apps/docs/tools/manual.mjs` generates between the BEGIN AUTOGEN and END AUTOGEN marks list every plugin and are left out. The check lists the addresses of the pages on docs.numen.md.

`network`: whether the plugin makes requests that can reach the internet, read statically from the code that is not a test, story, fixture or test data, and from the manifest; calls made inside dependencies are not traced. A request to the app's own server or to the person's own machine is not a network request. It lists:

- the Go client calls: `http.Client`, `http.Transport`, `http.DefaultClient`, `http.Get`, `http.Head`, `http.Post`, `http.PostForm`, `http.NewRequest`, `net.Dial`, `net.Dialer`, `tls.Dial`, `websocket.Dial` and `grpc.Dial` or `grpc.NewClient` (a plugin that only serves HTTP makes no request). The calls of a file count unless every `http://`, `https://`, `ws://` or `wss://` literal and every `localhost:`, `127.x.x.x:` or `[::1]:` port literal of that file names a loopback host; a file with no such literal calls an address it is given (a server the person configures, as in connections-llm), so its calls count;
- the frontend calls under `frontend/`: `fetch(`, `new XMLHttpRequest`, `new WebSocket(` and `new EventSource(`. A call counts when its target is a string or template literal that starts with an absolute `http(s)://`, `ws(s)://` or `//` URL of a host that is not loopback, and that host is listed. A relative path (`/…`, `./…`), a URL built by `getAssetUrl(…)` of `@numen/plugin-sdk` and a URL of `localhost`, `*.localhost`, `127.x.x.x`, `[::1]` or `0.0.0.0` do not count. A call whose target is not a literal (a variable, or a template literal that starts with `${`) counts when the plugin's frontend code names a host in a URL literal, and does not otherwise;
- the hosts of the `http://` and `https://` string literals of that code, without IP addresses, `localhost`, the reserved names (`.test`, `.example`, `.invalid`, `.localhost`, `.local`, `example.com`, `example.org`, `example.net`) and the XML namespace hosts `www.w3.org`, `purl.org`, `www.idpf.org` and `www.daisy.org`;
- the `frameOrigins` of the manifest and the hosts its `dependencies` are downloaded from;
- "Opens links in your browser" for the `link:open` permission: the link opens in the person's browser, not in the plugin, so it is no request of the plugin.

The answer is yes when there is a counted client call, a framed origin or a download.

For an external plugin, the source is cloned at the tag of its release; where git is not installed or the clone fails, `stories` and `network` are left out, the vulnerabilities are not scanned and no dependency is listed.

## Building it locally

```
node tools/plugin-registry/collect.mjs --out .registry --offline
NUMEN_PLUGIN_REGISTRY_FILE=$PWD/.registry/latest.json npm run build --prefix apps/landing
```

`--offline` also skips the vulnerability scan and reads Go licenses from the module cache only. `--quality <dir>` reads `mutation.json` and `coverage.json` from a folder; `tools/plugin-registry/quality-from-run.mjs` writes that folder from the artifacts of a `mutation.yml` run and a `coverage.yml` run. Without NUMEN_PLUGIN_REGISTRY_FILE the site reads the published registry, and a registry it cannot read fails the build.
