# libs/plugin/catalog

The plugin catalog: the manifest model (`manifest`), its JSON reader (`manifestjson`) and the sources the catalog is read from.

## Dependency on the core protocol

`libs/plugin/catalog` may import `libs/core/protocol` (the library matrix in `tools/checks/structure/ts/libraries/library-matrix.json` allows it). Only the packages that read the wire form of a manifest use it: `manifestjson` decodes the generated `numen.v1` manifest messages into the types of `manifest`. No non-test file of `manifest` imports a protocol type, so the domain of the core, which imports `manifest`, stays free of generated code.
