/**
 * The relationships between notes as this window offers them to plugins: the wikilinks of a text,
 * resolved by the core. The window writes no relationships, so every other call is refused.
 */
import { resolveTextLinks, type VaultRelations } from '@numen/plugin-loader'
import type { WindowClients } from '@/shared/api/clients'

const refuse = (): Promise<never> =>
  Promise.reject(new Error('this window reads and writes no relationships between notes'))

/** The relationships offered to plugins, with the wikilinks resolved over `notes`. */
export const createWindowRelations = (notes: WindowClients['notes']): VaultRelations => ({
  roles: [],
  getConnection: refuse,
  saveConnection: refuse,
  removeSide: refuse,
  removeConnection: refuse,
  join: refuse,
  createRelated: refuse,
  listBacklinks: refuse,
  resolveLinks: (text, vault) => resolveTextLinks(notes, { text, vault }),
})
