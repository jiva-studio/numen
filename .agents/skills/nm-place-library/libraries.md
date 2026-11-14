# The libraries in detail

For each library: what it holds, what it may import, who may import it, and the test for "does this belong here". The import rows are those of the package matrix in `tools/checks/structure/ts/libraries/check.mjs`.

## `@numen/ui` (`libs/ui/components`)

Holds pure components of general purpose, the design tokens, and the pure helpers beneath the components: the clock and schedule (`Clock`, `browserClock`, `getNow`), the viewport, geometry and bounds, text cutting, number formatting, request guards and stream followers.

Imports Vue and third-party UI code (`reka-ui`, Tailwind, Lucide, `markdown-it`) and no other first-party package. Imported by `@numen/workbench`, `@numen/plugin-manager`, `@numen/plugin-sdk`, every window and every plugin frontend, through the main entry `@numen/ui` only.

Plugins import this library, so whatever a plugin needs from the shared UI is added here.

Test for "does this belong here": the component or function can be described without a domain word, takes only props or arguments it defines itself, and a second application could use it unchanged. A file that needs `@numen/protocol`, a wire type, a note, a vault, a card or a plugin does not belong.

A component that was written for one window and names its domain stays with that window. When a general part can be separated, it moves here with a vocabulary of general words (`rows`, `items`, `at`) and the window maps its domain onto it.

## `@numen/workbench` (`libs/ui/workbench`)

Holds the chrome every desktop window shares: the workspace of panes and tabs, the command palette, the notice queue, the window frame and its platform insets.

Imports `@numen/ui`, `reka-ui` and Vue, nothing else. Imported by the windows.

Test: every desktop window draws it the same way and it needs no knowledge of what a tab shows.  It holds what only applications need; a plugin never imports it, and a component a plugin needs is added to `@numen/ui` instead. Behavior of one window (what its tabs contain, what its commands do) stays in that window.

## `@numen/core-client` (`libs/core/client`)

Holds the service clients, the transport, the enum codes the windows word, the error sentences, the manifest reader, the catalog client, a `testing` entry with connection doubles, and the re-exported generated types the applications name. Private; never published.

Imports `@numen/protocol` and `@connectrpc`. Imported by the windows and `@numen/plugin-loader`. No component library, no plugin and no composable outside an `api` segment imports it.

Test: it calls the core, or words what the core returned. A function that does not touch the wire or the schema belongs elsewhere. A client takes its transport as a parameter with the shared transport as default, so a test passes its own.

## `@numen/protocol` (`libs/core/protocol`)

Holds the `.proto` files, the committed generated Go (`*.pb.go`) and TypeScript (`*_pb.ts`), and the JSON declarations the generators read (`commands.json`, `events.json`, `slots.json`, `when.json`).

Imports nothing of this repository. Declared only by `@numen/plugin-sdk` and `@numen/core-client`; windows reach the types through `@numen/core-client`, plugins through the SDK.

Test: it is data that crosses the wire between Go and TypeScript. One service holds one subject, named in one word; a method is a verb and a noun (`GetSettings`, `ListReviewDays`); a reference field is named after the message it points at; an enum's zero value is documented. Generated code is never edited; `make generate-check` fails on a stale tree.

## `@numen/plugin-manager` (`libs/plugin/manager`)

Holds the plugin list, the plugin page and the settings card, their view-model types (`PluginInfo`, `PluginListing`, `ManagedPlugin`), the query parser and filter, the plugin state words, and a `testing` entry with listings.

Imports `@numen/ui` and Vue. Imported by the windows. The window reads the catalog through `@numen/core-client` and maps catalog entries to `PluginListing`; the library never sees a wire message.

Test: it is about listing, filtering or showing plugins, and it takes data by props and answers by events.

## `@numen/plugin-loader` (`libs/plugin/loader`)

Holds the loader of plugin frontends, the view, slot, drop and style registries, the keybinding and `when` readers, and the host side of the plugin contract (connections, text links, texts) over the core client. Each application supplies a `LoaderHost`.

Imports `@numen/plugin-sdk`, `@numen/core-client`, Vue and Vite; nothing of `@numen/ui`, `@numen/workbench`, `@numen/protocol` or an application. Imported by the windows; no plugin imports it.

Test: it makes a plugin frontend load, start, register what it contributes, or call the core on the plugin's behalf.

## `@numen/plugin-sdk` (`libs/plugin/sdk-ts`)

Holds what a plugin frontend is built from: the plugin entry, `createApi` for the plugin's own backend, views, slots, commands, notices, tasks, texts, the `testing` entry, the host-modules entry and the Vite helpers.

Imports `@numen/protocol` and `@numen/ui`; none of its exported declarations reaches a declaration of `libs/core/protocol`. Imported by plugins and `@numen/plugin-loader`.

Test: at least one plugin imports it, or it is the contract of a capability (agent provider, embedding). It holds no concrete agent, engine or runtime, and no application-side API. A call or field added here is added to `libs/plugin/sdk-go` in the same change.

## A new frontend package under `libs/`

The package matrix, `all-ts-packages-eslint` and `every-path-has-a-layer` require a row, an ESLint configuration and a place before the package exists. Creating a package is a decision for the user; propose it with the two applications that need it.
