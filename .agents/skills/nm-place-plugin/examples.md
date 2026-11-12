# Worked examples

Each example shows the placement the skill states, in code that already follows it. Where a plugin in `plugins/` is cut differently, the example is not taken from there.

## A small backend: `plugins/tools-updater`

The plugin asks a release feed for the newest build, compares it with the running one and serves the result to its frontend.

```text
backend/
├── main.go            # server.Run(service.Mount)                                  main.go:9-11
├── service/service.go # Fetcher interface (:30), the handlers, mount (:149), Mount (:160)
├── api/api.go         # JSON messages: Status, Update, File, state words
├── release/           # what a release is and how it is compared (evaluate.go, manifest.go); imports strconv and strings only
└── feed/feed.go       # the HTTP adapter: fetches the manifest; imports release (:13)
```

- `service` declares `Fetcher` (`service/service.go:30`), the one interface it needs from the outside, and `Mount` builds the real adapter with `feed.New()` (`:160-162`). A test calls `mount` with a fake `Fetcher` (`:149`).
- `release` is the domain. `release/evaluate.go:3-6` imports `strconv` and `strings`; the rules of comparing builds are testable with no network.
- `feed` is the adapter to the machine. It imports the domain it fills (`feed/feed.go:13`) and nothing of `service`.
- A handler (`updater.check`, `service.go`) decodes `api.CheckRequest`, validates it into a domain value (`newCurrent`), asks the adapter, and converts the domain result into `api.Status` (`newStatus`, `newUpdate`).
- The permission is `link:open` (`manifest.json:17`, consent to open external links; it does not gate the requests of the backend), the only host capability the plugin uses; it reads no vault.
- The setting `channel` and `endpoint` are read in `SetUpdateConfig` (`service.go:151`), and the manifest declares them.

Placing a new feature here: a new rule about builds goes in `release`; a new source of manifests goes in a new adapter package (or `feed`) behind `Fetcher`; a new message goes in `api` and in the fixtures of `testdata/api/`.

## A backend over the vault: `plugins/views-flashcards-studio`

The plugin runs scenarios over notes in a vault: cards, decks, review, schedules.

- `backend/ports/ports.go` declares what the scenarios require, in their vocabulary: `Clock` (`:32`), `VaultReader` (`:37`), and the writers, queries and stores after it. The package comment states that a use case does not depend on how the host implements them.
- `backend/cards`, `backend/flashcards` and `backend/review` are the scenario and domain packages; `format`, `note` and `cardid` hold what a card and a note are.
- `backend/hostports/hostports.go:21-45` builds every port from one function giving the narrow vault interfaces per vault identity and one function giving the plugin folder per vault identity. It is the adapter to the host; the target is to place it at `backend/internal/adapter/host/`, since it is an adapter named after the technology it uses.
- `service/` holds the handlers of the JSON methods; `backend/wire/` holds the conversions to `api` (`cardwire`, `presetwire`, `reviewwire`, `filewire`, `apierror`) and the command and tool bindings (`cardcommands`, `cardtools`).
- `service` is the only package that calls `hostports.New`, so a use case is tested with a fake port.

Placing a new feature here: a new question the scenarios ask of the vault is a method of a port in `ports`, answered in the host adapter, with a fake in the tests; a new scenario is a function of a scenario package that takes ports.

## Host clients and closing: `plugins/readers-pdf/backend/service/mount.go`

- `Mount` (`:14-33`) builds each host client from the environment, once: `host.NewTextClientFromEnv`, `NewStoreClientFromEnv`, `NewClientFromEnv`, `NewQueueClientFromEnv`.
- It passes them to the servers as narrow SDK interfaces such as `host.FileReader` and `host.TextVersions` (`mount`, `:35`), so `hosttest` fakes replace them in tests.
- It registers the API (`s.NewAPI(APIService)`), the page route, the commands (`command.Mount`) and what to close (`s.OnClose`).
- The engine is the package `backend/pdf`, outside `service`. `service` calls it, and `pdf` imports nothing of `service`.
- The manifest declares `vault:read` and `vault:write` (`manifest.json:50`), the permissions the vault calls need; the host rejects a call of a backend without them in `hostendpoint.authorize` (`libs/core/kernel/adapter/plugin/hostendpoint/host_endpoint.go:185`).

Placing a new feature here: a new host call is a client built in `Mount` and passed down as an interface; a new permission is added in the manifest in the change that adds the call.

## A frontend and its segments: `plugins/tools-updater/frontend`

```text
frontend/
├── index.ts          # builds UpdateParts (api), creates the model, registers the view   index.ts:19-29
├── api/updater.ts    # the JSON messages and SERVICE, matching backend/api
├── api/wire.ts       # createApi(SERVICE) behind the UpdateApi interface
├── model/updates.ts  # createUpdates(deps): state and actions; imports UpdateApi as a type only   updates.ts:8-11
├── lib/release.ts    # pure: formatReleaseDate, getPlatformMessage
├── UpdateIndicator.vue, UpdatePopover.vue   # the humble views
└── words.ts, types.ts, styles.css
```

- The direction is `ui` to `model` to `api` to `lib`: `model/updates.ts` imports the `UpdateApi` type of `api/wire` (`:10`) and the message types of `api/updater` (`:11`); `lib/release.ts` imports `../types` and `../words` and none of the segments.
- `index.ts` builds `UpdateParts` (`:19-29`) with the real `updater`, and `createUpdates` takes them as `deps`, so `index.test.ts` and `model/updates.test.ts` pass fakes.
- `api/updater.test.ts` reads the JSON files of `testdata/api/`, which the Go tests of `backend/api` read as well. The frontend never imports a Go package, and the backend never imports a TypeScript file.
- The plugin imports `@numen/plugin-sdk` and `@numen/ui` and nothing else of the repository (`package.json`, `dependencies`).

For a slice inside a larger frontend, `plugins/readers-pdf/frontend/reader` has `lib/hand.ts`, `model/scroll.ts` importing `../lib/hand` (`model/scroll.ts:9`), and `ui/Reader.vue`; the direction is the same.

Placing a new feature here: a new calculation goes in `lib` with its test, a new piece of state in `model`, a new backend call in `api` (type, wire method, fixture), and a new component in `ui`; `index.ts` changes only when a registration or a real part is added.
