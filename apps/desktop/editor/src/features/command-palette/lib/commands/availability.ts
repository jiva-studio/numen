/**
 * When a command is offered, and when it is not.
 *
 * A command is offered for what is currently selected: a note or a vault.
 * Each function here answers one such question, and the table in `table.ts` names
 * which one a command uses.
 */
import type { CommandTarget } from '../../types'

/** A command over the note in front, which there has to be one of. */
export const isNoteOpen = (target: CommandTarget): boolean => target.isReady && target.path !== ''

/** A command over the vault in front, which there has to be one of. */
export const isVaultOpen = (target: CommandTarget): boolean => target.vault.id !== ''

export const isAlwaysAvailable = (): boolean => true
