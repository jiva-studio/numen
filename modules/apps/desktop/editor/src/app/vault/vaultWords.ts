/**
 * Vault domain conversions between schema representations and window models.
 */
import {
  FlushResult,
  Presence as Presences,
  SearchMode as Modes,
  Unit as Units,
  VaultsRefusal,
} from '@numen/protocol'
import type {
  Vault as VaultMessage,
} from '@numen/protocol'
import type { TallyUnit } from '@numen/ui'
import { namesOf } from '@numen/wire'
import type { SearchMode } from '@/features/command-palette'
import type { Presence } from '@/entities/settings'
import type { Vault, VaultErrorCode, VaultResult } from '@/shared/vaults'

/**
 * What a piece of work counts, in the words the window uses. One it has no word
 * for is counted one by one.
 */
export const counted: Record<Units, TallyUnit> = {
  [Units.UNSPECIFIED]: 'things',
  [Units.THINGS]: 'things',
  [Units.BYTES]: 'bytes',
  [Units.SECONDS]: 'seconds',
}

/**
 * How a search is asked, in the words the window uses.
 */
const asked: Record<Modes, SearchMode | null> = {
  [Modes.UNSPECIFIED]: null,
  [Modes.HYBRID]: 'hybrid',
  [Modes.WORDS]: 'words',
  [Modes.MEANING]: 'meaning',
  [Modes.NAMES]: 'names',
}

/** How a search is asked, as the schema names it. */
export const modes = namesOf<SearchMode, Modes>(asked)

/** What a model's files are on this machine, in the words the window uses. */
export const fetched: Record<Presences, Presence> = {
  [Presences.UNSPECIFIED]: 'nothing to fetch',
  [Presences.PRESENT]: 'present',
  [Presences.NOT_FETCHED]: 'not fetched',
  [Presences.NOTHING_TO_FETCH]: 'nothing to fetch',
}

const unvaulted: Record<VaultsRefusal, VaultErrorCode> = {
  [VaultsRefusal.UNSPECIFIED]: 'unreadable',
  [VaultsRefusal.UNREADABLE]: 'unreadable',
  [VaultsRefusal.COPY]: 'copy',
  [VaultsRefusal.OVERLAPS]: 'overlaps',
  [VaultsRefusal.NAME_TAKEN]: 'nameTaken',
  [VaultsRefusal.LAST_VAULT]: 'lastVault',
  [VaultsRefusal.SHOWING]: 'showing',
  [VaultsRefusal.UNKNOWN]: 'unknown',
  [VaultsRefusal.NO_TRASH]: 'noTrash',
  [VaultsRefusal.ASKING]: 'asking',
}

export const getVaultError = (from: { error?: VaultsRefusal | undefined; refusal?: VaultsRefusal | undefined }): VaultErrorCode | null => {
  const code = from.error ?? from.refusal
  return code === undefined ? null : unvaulted[code]
}

/**
 * What a client has left, in the words the window uses.
 */
const left: Record<FlushResult, 'nothing' | 'written' | 'asking' | null> = {
  [FlushResult.UNSPECIFIED]: null,
  [FlushResult.NOTHING]: 'nothing',
  [FlushResult.WRITTEN]: 'written',
  [FlushResult.ASKING]: 'asking',
}

export const owing = namesOf<NonNullable<(typeof left)[FlushResult]>, FlushResult>(left)

/** One vault of the list, kept as the plain value the window carries it as. */
export const mapVault = (one: VaultMessage): Vault => ({
  id: one.id,
  name: one.name,
  path: one.path,
  missing: one.missing,
})

export const mapVaultResult = (from: {
  vault?: VaultMessage | undefined
  error?: VaultsRefusal | undefined
  refusal?: VaultsRefusal | undefined
}): VaultResult => {
  const error = getVaultError(from)
  return {
    vault: from.vault ? mapVault(from.vault) : null,
    error,
  }
}
