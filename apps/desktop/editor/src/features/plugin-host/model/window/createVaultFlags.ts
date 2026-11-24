import { reactive } from 'vue'
import type { JsonObject, VaultFlagsHost } from '@numen/plugin-loader'

/** The flags plugins set on the vaults, which the when conditions read as `vault.flags`. */
export interface VaultFlags {
  getFlags: (vaultId: string) => JsonObject
  createHost: (pluginId: string) => VaultFlagsHost
}

type Branch = Record<string, unknown>

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
