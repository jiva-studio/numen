import { reactive } from 'vue'
import type { JsonObject, VaultFlagsHost } from '@numen/plugin-loader'

/** The flags plugins set on the vaults, which the when conditions read as `vault.flags`. */
export interface VaultFlags {
  /** The flags of a vault nested by the dotted id of the plugin and the key; empty for a vault none is set on. */
  getFlags: (vaultId: string) => JsonObject
  /** What one plugin sets its flags with, under its own id. */
  createHost: (pluginId: string) => VaultFlagsHost
}

type Branch = Record<string, unknown>

/** The branch of the name under the node, made as it is first needed; undefined where a flag stands in the way. */
const getBranch = (node: Branch | undefined, name: string): Branch | undefined => {
  if (node === undefined) return undefined
  node[name] ??= Object.create(null) as Branch
  const branch = node[name]
  return typeof branch === 'object' ? (branch as Branch) : undefined
}

const nest = (flags: Readonly<Record<string, boolean>>): JsonObject => {
  const root = Object.create(null) as Branch
  for (const [name, value] of Object.entries(flags)) {
    const path = name.split('.')
    const last = path.pop() ?? name
    const node = path.reduce<Branch | undefined>(getBranch, root)
    if (node !== undefined && !(last in node)) node[last] = value
  }
  return root as JsonObject
}

export function createVaultFlags(): VaultFlags {
  const flags = reactive<Record<string, Record<string, boolean>>>({})
  return {
    getFlags: (vaultId) => nest(flags[vaultId] ?? {}),
    createHost: (pluginId) => ({
      setFlag: (vaultId, key, value) => {
        flags[vaultId] = { ...flags[vaultId], [`${pluginId}.${key}`]: value }
      },
    }),
  }
}
