/**
 * Vault domain conversions between schema representations and window models.
 */
import type { TallyUnit } from '@numen/shell'
import {
  asFailure,
  asValue,
  COMPLETION_CONSENT,
  FLUSH_RESULT,
  getNames,
  MODEL_PRESENCE,
  SEARCH_MODE,
  TASK_UNIT,
  VAULTS_ERROR_CODE,
  type CompletionConsentCode,
  type FlushResultCode,
  type ModelPresenceCode,
  type SearchModeCode,
  type TaskUnitCode,
  type VaultsErrorCodeValue,
} from '@numen/plugin-sdk'
import type { SearchMode } from '@/features/command-palette'
import type { Consent, Presence } from '@/entities/settings'
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
export const counted: Record<TaskUnitCode, TallyUnit> = {
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

/** What a model's files are on this machine, in the words the window uses. */
export const fetched: Record<ModelPresenceCode, Presence> = {
  [MODEL_PRESENCE.UNSPECIFIED]: 'nothing to fetch',
  [MODEL_PRESENCE.PRESENT]: 'present',
  [MODEL_PRESENCE.NOT_FETCHED]: 'not fetched',
  [MODEL_PRESENCE.NOTHING_TO_FETCH]: 'nothing to fetch',
}

const unvaulted: Record<VaultsErrorCodeValue, VaultErrorCode> = {
  [VAULTS_ERROR_CODE.UNSPECIFIED]: 'unreadable',
  [VAULTS_ERROR_CODE.UNREADABLE]: 'unreadable',
  [VAULTS_ERROR_CODE.COPY]: 'copy',
  [VAULTS_ERROR_CODE.OVERLAPS]: 'overlaps',
  [VAULTS_ERROR_CODE.NAME_TAKEN]: 'nameTaken',
  [VAULTS_ERROR_CODE.LAST_VAULT]: 'lastVault',
  [VAULTS_ERROR_CODE.SHOWING]: 'showing',
  [VAULTS_ERROR_CODE.UNKNOWN]: 'unknown',
  [VAULTS_ERROR_CODE.NO_TRASH]: 'noTrash',
  [VAULTS_ERROR_CODE.ASKING]: 'asking',
}

export const getVaultError = (from: {
  error?: VaultsErrorCodeValue | undefined
  // Stryker disable next-line ConditionalExpression: indexing unvaulted with undefined yields null via fallback either way.
}): VaultErrorCode | null => (from.error === undefined ? null : (unvaulted[from.error] ?? null))

/**
 * What a client has left, in the words the window uses.
 */
const flushResultNames: Record<FlushResultCode, 'nothing' | 'written' | 'asking' | null> = {
  [FLUSH_RESULT.UNSPECIFIED]: null,
  [FLUSH_RESULT.NOTHING]: 'nothing',
  [FLUSH_RESULT.WRITTEN]: 'written',
  [FLUSH_RESULT.ASKING]: 'asking',
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

/** Whether the user allows a plugin to query a model, in the words the window uses. */
export const consented: Record<CompletionConsentCode, Consent> = {
  [COMPLETION_CONSENT.UNSPECIFIED]: 'none',
  [COMPLETION_CONSENT.PENDING]: 'pending',
  [COMPLETION_CONSENT.REQUESTED]: 'requested',
  [COMPLETION_CONSENT.ALLOWED]: 'allowed',
  [COMPLETION_CONSENT.DENIED]: 'denied',
}
