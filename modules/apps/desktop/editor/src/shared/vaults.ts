/**
 * The vaults an installation holds, and what the window asks of the list.
 */

/** One vault the installation holds, as the list has it. */
export interface Vault {
  /** The identity the folder carries, and how the vault is asked for again. */
  readonly id: string
  /** What the person calls the collection. */
  readonly name: string
  /** The folder, absolute on this machine. */
  readonly path: string
  /** Whether nothing is at the path. The vault stays on the list. */
  readonly missing: boolean
}

/** Every vault the installation holds, and the one this window is showing. */
export interface VaultList {
  readonly vaults: readonly Vault[]
  /** The identity of the vault in front of the person. */
  readonly showing: string
}

/** What adding a vault came back with, and what renaming one comes back with. */
export interface VaultResult {
  /** The vault as the list has it now. Null where the list is as it was. */
  vault: Vault | null
  error?: VaultErrorCode | null
  refusal: VaultRefusalReason | null
}

/** Why the list is as it was, or why the window is showing what it was showing. */
export type VaultErrorCode =
  | 'unreadable'
  | 'copy'
  | 'overlaps'
  | 'nameTaken'
  | 'lastVault'
  | 'showing'
  | 'unknown'
  | 'noTrash'
  | 'asking'
export type VaultRefusalReason = VaultErrorCode

/** The vaults an installation holds, and what changes them. */
export interface Vaults {
  /** Every vault on the list, and which of them this window is showing. */
  list(): Promise<VaultList>
  /**
   * This machine's own folder picker, put in front of the person. It answers
   * with the folder they chose, and with nothing where they closed it.
   */
  choose(title: string): Promise<string>
  /**
   * A folder turned into a vault and put on the list. The folder is given an
   * identity that stays with it wherever it moves to.
   */
  add(path: string, name: string): Promise<VaultResult>
  /** What a person calls a vault. The folder keeps the name the filesystem gives it. */
  rename(id: string, name: string): Promise<VaultResult>
  /**
   * A vault taken off the list. The folder stays where it is, and goes to the
   * trash this machine keeps when the call asks for it.
   */
  remove(id: string, trash: boolean): Promise<VaultRefusalReason | null>
  /** Another vault shown in this window, in place of the one it was showing. */
  open(id: string): Promise<VaultRefusalReason | null>
}
