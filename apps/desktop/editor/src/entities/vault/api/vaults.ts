/**
 * The vaults the installation holds, and the operations on the list.
 */
import { VAULTS_ERROR_CODE, type VaultsErrorCodeValue } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import type { Vault, VaultAction, VaultErrorCode, VaultResult, Vaults } from '../lib/vault'

interface VaultWire {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly isMissing: boolean
}

const unvaulted: Record<VaultsErrorCodeValue, VaultErrorCode> = {
  [VAULTS_ERROR_CODE.UNSPECIFIED]: 'unreadable',
  [VAULTS_ERROR_CODE.UNREADABLE]: 'unreadable',
  [VAULTS_ERROR_CODE.COPY]: 'copy',
  [VAULTS_ERROR_CODE.OVERLAPS]: 'overlaps',
  [VAULTS_ERROR_CODE.NAME_TAKEN]: 'nameTaken',
  [VAULTS_ERROR_CODE.LAST_VAULT]: 'lastVault',
  [VAULTS_ERROR_CODE.CURRENT]: 'current',
  [VAULTS_ERROR_CODE.NOT_FOUND]: 'notFound',
  [VAULTS_ERROR_CODE.NO_TRASH]: 'noTrash',
  [VAULTS_ERROR_CODE.UNSAVED]: 'unsaved',
}

const getVaultError = (from: {
  error?: VaultsErrorCodeValue | undefined
}): VaultErrorCode | null => (from.error === undefined ? null : unvaulted[from.error])

const mapVault = (one: VaultWire): Vault => ({
  id: one.id,
  name: one.name,
  path: one.path,
  isMissing: one.isMissing,
})

const mapVaultResult = (from: {
  vault?: VaultWire | undefined
  error?: VaultsErrorCodeValue | undefined
}): VaultResult => {
  const error = getVaultError(from)
  if (error) return { ok: false, error }
  return { ok: true, value: from.vault ? mapVault(from.vault) : null }
}

const mapNoValue = (from: { error?: VaultsErrorCodeValue | undefined }): VaultAction => {
  const error = getVaultError(from)
  return error ? { ok: false, error } : { ok: true, value: null }
}

/**
 * The vaults this installation holds over `vaultsService`, in the shape the window requests them.
 */
export const createVaults = (vaultsService: WindowClients['vaultsService']): Vaults => ({
  list: async () => (await vaultsService.listVaults({})).vaults.map(mapVault),
  choose: async (title) => {
    const answer = await vaultsService.chooseFolder({ title, startingAt: '' })
    return answer.isChosen ? answer.path : ''
  },
  add: async (path, called) => mapVaultResult(await vaultsService.addVault({ path, name: called })),
  rename: async (id, called) =>
    mapVaultResult(await vaultsService.renameVault({ id, name: called })),
  remove: async (id, trash) => mapNoValue(await vaultsService.removeVault({ id, trash })),
  open: async (id) => mapNoValue(await vaultsService.openVault({ id })),
})
