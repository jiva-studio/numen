/**
 * The words the palette is drawn in: every command as it is offered, every
 * that has been carried out answers in.
 *
 * Nothing here is a decision. `types.ts` says what a command is and
 * `lib/table.ts` which commands there are; this is only what a person reads of
 * them.
 */
// Stryker disable StringLiteral: interface copy
import type { PaletteKeys } from '@numen/ui'
import { VAULT_ERRORS } from '@/entities/vault'
import type { VaultErrorCode } from '@/entities/vault'
import type { ErrorCode } from '@/shared/errors'
import { WORDS } from '@/shared/words'
import type { EmptyWords } from './model/search/search'

export const ANSWER_WORDS: AnswerWords = {
  ...WORDS,
  vaultErrors: VAULT_ERRORS,
}

/** Everything carrying a command out says in the window's voice. */
export interface AnswerWords {
  /** What the vault reported as error, in words a person reads. */
  readonly errors: Record<ErrorCode, string>
  /** What the list of vaults reported as error, in words a person reads. */
  readonly vaultErrors: Record<VaultErrorCode, string>
  /** What the machine's own folder picker is titled. */
  readonly folder: string
  /** The links that reach nothing now, which nothing repairs. */
  readonly dangling: string
  /** The vault opens with no note at all. */
  readonly nowhere: string
  /** The note is waiting on the person, and its file stays where it is. */
  readonly unanswered: string
  /** The note holds prose nobody here has seen, so nothing was written. */
  readonly stale: string
  /** A name at the destination is taken, and the file stayed where it was. */
  readonly occupied: string
}

/** The words the step that asks for the name typed back is drawn in. */
export interface RetypeWords {
  readonly action: string
  readonly detail: string
  readonly back: string
}

/** The words the step that confirms is drawn in. */
export interface ConfirmWords {
  /** The answer that changes nothing, and what it leaves. */
  readonly keeps: string
  readonly retainedNotice: string
  /** The answer that does it, and what it leaves. */
  readonly action: string
  readonly detail: string
}

/** Everything the commands say in the window's voice. */
export interface Words extends EmptyWords {
  /** The commands, each in the words it is offered by. */
  readonly read: string
  readonly beside: string
  readonly travel: string
  readonly child: string
  readonly parent: string
  readonly jump: string
  readonly title: string
  readonly remove: string
  readonly destroy: string
  readonly copy: string
  readonly newNote: string
  /** The note that points at a web address, made from the address. */
  readonly importUrl: string
  readonly close: string
  /**
   * The four commands over how the window is drawn: the theme, the halves, and
   * the two sizes.
   */
  readonly appearance: string
  readonly mode: string
  readonly interfaceScale: string
  readonly textScale: string
  /** The command over whether a note's title and its filename are one name. */
  readonly syncing: string
  /** The command over whether a node nests the headings of its note under it. */
  readonly nesting: string
  /** The command over how many of them stand under a node at once. */
  readonly headings: string
  /** The command over the performance workload profile. */
  readonly performanceProfile: string
  /** Everything this installation is configured as, in a tab of its own. */
  readonly settings: string
  readonly find: string
  /** The keystroke the search answers to away from the palette. */
  readonly findKeys: PaletteKeys
  readonly first: string
  readonly goto: string
  /** The commands over the vaults this installation holds. */
  readonly openVault: string
  readonly newVault: string
  readonly renameVault: string
  readonly removeVault: string
  readonly eraseVault: string
  /** The groups the commands are drawn in. */
  readonly noteGroupTitle: string
  readonly overFile: string
  readonly overWindow: string
  readonly overVault: string
  /** Why nothing can be done to a note: the vault never opened, or none is in front. */
  readonly noVault: string
  readonly noNote: string
  readonly command: string
  readonly typeCommand: string
  /** A name asked for, and the one item it offers. */
  readonly naming: string
  readonly typeName: string
  readonly callIt: string
  /** A value a command asks for, and the two things its step offers. */
  readonly givesIt: string
  readonly leavesOut: string
  /**
   * what is said of words that are no address.
   */
  readonly url: string
  readonly address?: string
  readonly typeAddress: string
  readonly importIt: string
  readonly notAnAddress: string
  /** A note asked for, over the names in the vault. */
  readonly names: string
  readonly typeNote: string
  /**
   * One of a list the window holds: the field, and what Enter does. The groups
   * such a list is drawn in are named by whatever holds it.
   */
  readonly typeChoice: string
  readonly chooses: string
  /** A vault asked for, over the vaults the installation holds. */
  readonly vaults: string
  readonly typeVault: string
  /** The two vaults the list draws and does not offer to choose. */
  readonly gone: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The two answers to the confirmation: the one that changes nothing, first. */
  readonly asking: string
  /** How many files a command is over, where it is over several. */
  readonly several: (files: number) => string
  readonly answer: string
  readonly retainedNotice: string
  /** The two answers over a vault, whose folder is left where it is. */
  readonly keepsVault: string
  readonly removeVaultAction: string
  readonly stays: string
  /** The name typed back, which is what destroying asks for. */
  readonly exactly: string
  readonly typeBack: string
  readonly destroys: string
  readonly forever: string
  /** The same, over a vault whose folder goes to the trash this machine keeps. */
  readonly typeVaultBack: string
  readonly erases: string
  readonly binned: string
  /** The note the vault could not find, offered as one to make. */
  readonly creating: string
  readonly creates: string
  /** The relations it can be made in, off the note in front. */
  readonly asChild: string
  readonly asParent: string
  readonly asJump: string
}

/** What the menu on a row of the files tree offers besides the commands over a note. */
export const PATH_WORDS = {
  remove: 'Remove',
  relation: 'Relation',
}
