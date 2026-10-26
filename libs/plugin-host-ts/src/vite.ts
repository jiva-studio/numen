/**
 * The build of the host modules an application serves to plugin frontends: one entry for each
 * module in `HOST_MODULES`, re-exporting it and written to its path there.
 */
import type { Plugin } from 'vite'
import { HOST_MODULES } from '@numen/plugin-sdk/host-modules'

/** The id of the entry that serves one host module to plugin bundles. */
const HOST_MODULE_ENTRY = 'host-module:'

/** Each host module as an entry that re-exports it. */
export function createHostModuleEntries(): Plugin {
  return {
    name: 'host-module-entries',
    resolveId: (id) => (id.startsWith(HOST_MODULE_ENTRY) ? `\0${id}` : undefined),
    load: (id) =>
      id.startsWith(`\0${HOST_MODULE_ENTRY}`)
        ? `export * from '${id.slice(HOST_MODULE_ENTRY.length + 1)}'`
        : undefined,
  }
}

/** The name of each host module's entry: its path without the leading slash and `.js`. */
export const hostModuleInputs: Readonly<Record<string, string>> = Object.fromEntries(
  Object.entries(HOST_MODULES).map(([name, path]) => [
    path.slice(1, -'.js'.length),
    HOST_MODULE_ENTRY + name,
  ]),
)

/** The file a built chunk is written to: a host module at its path, everything else hashed. */
export function getEntryFileName(chunk: { readonly name: string }): string {
  return chunk.name in hostModuleInputs ? '[name].js' : 'built/[name]-[hash].js'
}
