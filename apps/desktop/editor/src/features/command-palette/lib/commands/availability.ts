/**
 * When a command is offered, and when it is not.
 *
 * A command is offered over what is in front of the person: a note or a vault.
 * Each answer here is one such question, and the table in `table.ts` names
 * which it asks.
 */
import type { CommandTarget } from '../../types'

/** A command over the note in front, which there has to be one of. */
export const isNoteOpen = (target: CommandTarget): boolean => target.isReady && target.path !== ''

/** A command over the vault in front, which there has to be one of. */
export const isVaultOpen = (target: CommandTarget): boolean => target.vault.id !== ''

export const isAlwaysOffered = (): boolean => true
