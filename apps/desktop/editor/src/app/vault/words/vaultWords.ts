/**
 * Vault domain conversions between schema representations and window models.
 */
import type { TallyUnit } from '@numen/workbench'
import {
  FLUSH_RESULT,
  getNames,
  SEARCH_MODE,
  TASK_UNIT,
  VAULTS_ERROR_CODE,
  type FlushResultCode,
  type SearchModeCode,
  type VaultsErrorCodeValue,
  asFailure,
  asValue,
} from '@numen/core-client'
import type { SearchMode } from '@/features/command-palette'
import type { Vault, VaultErrorCode, VaultResult } from '@/entities/vault'

interface VaultWire {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly isMissing: boolean
}

/**
 * What a piece of work counts, in the words the window uses. One it has no word
 * for is counted one by one.
 */
export const counted: Partial<Record<number, TallyUnit>> = {
  [TASK_UNIT.UNSPECIFIED]: 'items',
  [TASK_UNIT.THINGS]: 'items',
  [TASK_UNIT.BYTES]: 'bytes',
  [TASK_UNIT.SECONDS]: 'seconds',
}

/**
 * Search modes, named as the window uses them.
 */
const query: Record<SearchModeCode, SearchMode | null> = {
  [SEARCH_MODE.UNSPECIFIED]: null,
  [SEARCH_MODE.HYBRID]: 'hybrid',
  [SEARCH_MODE.WORDS]: 'words',
  [SEARCH_MODE.MEANING]: 'meaning',
  [SEARCH_MODE.NAMES]: 'names',
}

/** Search modes, as the schema names them. */
export const modes = getNames<SearchMode, SearchModeCode>(query)

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

export const getVaultError = (from: {
  error?: VaultsErrorCodeValue | undefined
}): VaultErrorCode | null => (from.error === undefined ? null : unvaulted[from.error])

/**
 * What a client has left, in the words the window uses.
 */
const flushResultNames: Record<FlushResultCode, 'nothing' | 'written' | 'unsaved' | null> = {
  [FLUSH_RESULT.UNSPECIFIED]: null,
  [FLUSH_RESULT.NOTHING]: 'nothing',
  [FLUSH_RESULT.WRITTEN]: 'written',
  [FLUSH_RESULT.UNSAVED]: 'unsaved',
}

export const flushResults = getNames<
  NonNullable<(typeof flushResultNames)[FlushResultCode]>,
  FlushResultCode
>(flushResultNames)

/** One vault of the list, kept as the plain value the window carries it as. */
export const mapVault = (one: VaultWire): Vault => ({
  id: one.id,
  name: one.name,
  path: one.path,
  isMissing: one.isMissing,
})

export const mapVaultResult = (from: {
  vault?: VaultWire | undefined
  error?: VaultsErrorCodeValue | undefined
}): VaultResult => {
  const error = getVaultError(from)
  return error ? asFailure(error) : asValue(from.vault ? mapVault(from.vault) : null)
}
