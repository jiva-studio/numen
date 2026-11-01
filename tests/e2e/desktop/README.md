# Desktop end-to-end tests

`npm test` (and `npm run test:flashcards`) runs Playwright against the debug server of `scripts/start-server.mjs` (`scripts/start-flashcards-server.mjs`): a temporary vault holds the config, cache, data and home folders, the plugins run on mock models with empty placeholder dependencies, and `scripts/stand-ins.mjs` serves the release manifest and the chat API on loopback.

The servers are the debug builds of the two windows, `apps/desktop/numen-debug` and `apps/desktop/numen-flashcards-debug`, built with the `numen_debug` tag (`go build -tags numen_debug -o numen-debug ./cmd/numen` in `apps/desktop`, and the same for `./cmd/numen-flashcards`); `make app_desktop_e2e` builds both.

On Linux `scripts/run-offline.mjs` runs the suite inside `unshare -rn` with loopback alone, so a scenario that connects to the internet or production fails; give it a local stand-in instead.

The window is WebKitGTK on Linux and WKWebView on macOS, so the suite runs in WebKit as well as Chromium. `npm test` runs the `chromium` project and then the `webkit` project, each with its own server and vault; `npm test -- --project=webkit` runs one. The reports are written to `playwright-report/<project>` (`npm run test:report -- playwright-report/webkit`) and the traces in `test-results/<project>`. WebKit on Linux uses glib-networking for https to the debug server: a distribution installs it where GLib looks for it, and on NixOS the runner takes it from nixpkgs and sets `GIO_EXTRA_MODULES`.

The debug server serves the window's page with `/numen-debug/fetch.js` loaded first, which sends the page's calls as PUT, and handles a PUT as POST. WebKit on Linux (libsoup) opens a connection for every POST, even over HTTP/2, and at most six to one host; the window keeps more server streams open than that, so without it every call after the sixth stream never completes. A PUT shares one HTTP/2 connection with everything else. The window itself uses `wails://`, where no connection is opened.
