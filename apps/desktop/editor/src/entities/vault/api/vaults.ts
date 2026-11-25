/**
 * The vaults the installation holds, and the operations on the list.
 */
import { WIRE_ERROR_CODE, type WireErrorCodeValue } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { Vault, VaultAction, VaultErrorCode, VaultResult, Vaults } from '../lib/vault'

interface VaultWire {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly isMissing: boolean
}

const unvaulted: Partial<Record<WireErrorCodeValue, VaultErrorCode>> = {
  [WIRE_ERROR_CODE.UNREADABLE]: 'unreadable',
  [WIRE_ERROR_CODE.VAULT_COPY]: 'copy',
  [WIRE_ERROR_CODE.VAULT_OVERLAPS]: 'overlaps',
  [WIRE_ERROR_CODE.ALREADY_EXISTS]: 'nameTaken',
  [WIRE_ERROR_CODE.LAST_VAULT]: 'lastVault',
  [WIRE_ERROR_CODE.VAULT_CURRENT]: 'current',
  [WIRE_ERROR_CODE.NOT_FOUND]: 'notFound',
  [WIRE_ERROR_CODE.NO_TRASH]: 'noTrash',
  [WIRE_ERROR_CODE.UNSAVED]: 'unsaved',
}

const getVaultError = (from: {
  error?: WireErrorCodeValue | undefined
}): VaultErrorCode | null =>
  from.error === undefined || from.error === WIRE_ERROR_CODE.UNSPECIFIED
    ? null
    : (unvaulted[from.error] ?? 'unreadable')

const mapVault = (one: VaultWire): Vault => ({
  id: one.id,
  name: one.name,
  path: one.path,
  isMissing: one.isMissing,
})

const mapVaultResult = (from: {
  vault?: VaultWire | undefined
  error?: WireErrorCodeValue | undefined
}): VaultResult => {
  const error = getVaultError(from)
  if (error) return { ok: false, error }
  return { ok: true, value: from.vault ? mapVault(from.vault) : null }
}

const mapNoValue = (from: { error?: WireErrorCodeValue | undefined }): VaultAction => {
  const error = getVaultError(from)
  return error ? { ok: false, error } : { ok: true, value: null }
}

/**
 * The vaults this installation holds over `vaultsService`, in the shape the window requests them, and
 * the folder dialog of this machine over `folderDialog`.
 */
export const createVaults = (
  vaultsService: WindowClients['vaultsService'],
  folderDialog: WindowClients['folderDialog'],
): Vaults => ({
  list: async () => (await vaultsService.listVaults({})).vaults.map(mapVault),
  choose: async (title) => {
    const answer = await folderDialog.chooseFolder({ title, startFolder: '' })
    return answer.isChosen ? answer.path : ''
  },
  add: async (path, called) => mapVaultResult(await vaultsService.addVault({ path, name: called })),
  rename: async (id, called) =>
    mapVaultResult(await vaultsService.renameVault({ id, name: called })),
  remove: async (id, trash) =>
    mapNoValue(await vaultsService.removeVault({ id, isFolderTrashed: trash })),
  open: async (id) => mapNoValue(await vaultsService.openVault({ id })),
})
