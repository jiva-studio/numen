/**
 * The vaults an installation holds, and the operations on the list.
 */
import type { Result } from '@numen/core-client'

/** One vault the installation holds, as the list has it. */
export interface Vault {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly isMissing: boolean
}

export interface VaultList {
  readonly vaults: readonly Vault[]
  readonly showing: string
}

/**
 * The result of adding or renaming a vault. A null value means the list is
 * unchanged, which is a valid outcome and not a failure.
 */
export type VaultResult = Result<Vault | null, VaultErrorCode>

export type VaultErrorCode =
  | 'unreadable'
  | 'copy'
  | 'overlaps'
  | 'nameTaken'
  | 'lastVault'
  | 'current'
  | 'notFound'
  | 'noTrash'
  | 'unsaved'

/** The vaults an installation holds, and what changes them. */
export interface Vaults {
  list: () => Promise<VaultList>
  /**
   * Opens the native folder picker. It resolves to the chosen folder, or to an
   * empty string when the picker is dismissed.
   */
  choose: (title: string) => Promise<string>
  /**
   * Adds a folder to the list as a vault. The folder is assigned an identity
   * that stays with it when it moves.
   */
  add: (path: string, name: string) => Promise<VaultResult>
  /** Sets the display name of a vault. The folder keeps its filesystem name. */
  rename: (id: string, name: string) => Promise<VaultResult>
  /**
   * Removes a vault from the list. The folder stays in place, and is moved to
   * the system trash when `trash` is true.
   */
  remove: (id: string, trash: boolean) => Promise<VaultErrorCode | null>
  /** Shows another vault in this window, replacing the current one. */
  open: (id: string) => Promise<VaultErrorCode | null>
}
