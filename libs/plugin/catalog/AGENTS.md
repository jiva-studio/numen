# libs/plugin/catalog

The plugin catalog: the manifest model (`manifest`), its JSON reader (`manifestjson`), the order of plugin versions (`semver`) and the sources the catalog is read from.

## Dependency on the core protocol

`libs/plugin/catalog` may import `libs/shared/protocol` (the library matrix in `tools/checks/structure/ts/libraries/library-matrix.json` allows it). `manifestjson` decodes the generated `numen.v1` manifest messages into the types of `manifest`. `manifest.ProtocolVersion` repeats `protocol.PluginProtocolVersion` of the root package, and a test of `manifestjson` keeps the two equal; no non-test file of `manifest` imports a protocol type or a generated package, so the domain of the core, which imports `manifest`, stays free of generated code.
