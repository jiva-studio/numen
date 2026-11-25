# tools/syncserver

The fake sync server of the tests: `fakeserver` (an in-process implementation of the sync protocol), `wiretest` and `slowwatchtest` (helpers for tests of the clients) and `cmd/fakesync` (the server as a command for the end-to-end and acceptance runs).

The module imports `libs/shared/syncprotocol` and `libs/shared/identity`; `libs/shared/syncprotocol` never imports it. Plugin tests reach it through `go.work`; no `go.mod` of a plugin requires it, and no production code imports it.

`clientcheck` holds the tests of `libs/shared/syncprotocol/client` that need the fake server.
