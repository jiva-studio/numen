/**
 * The words the palette is drawn in: every command as it is offered, every
 * group it stands in, what each step says while it asks, and what a command
 * that has been carried out answers in.
 *
 * Nothing here is a decision. `types.ts` says what a command is and
 * `lib/table.ts` which commands there are; this is only what a person reads of
 * them.
 */
import type { PaletteKeys } from '@numen/ui'
import { FETCHED, MADE } from '@/entities/artifact'
import type { Artifact, ArtifactState } from '@/entities/artifact'
import { VAULT_ERRORS } from '@/entities/vault'
import type { VaultErrorCode } from '@/entities/vault'
import type { ErrorCode } from '@/shared/errors'
import { WORDS } from '@/shared/words'
import type { EmptyWords } from './model/search'

/**
 * What a command answers in: the window's own voice, the words a run over a
 * file is spoken about in, and the words the list of vaults reports in. Each is
 * written where it belongs, and a command is where they meet.
 */
export const ANSWER_WORDS: AnswerWords = {
  ...WORDS,
  made: MADE,
  fetched: FETCHED,
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
  /** This build cannot do the run at all, and stops offering it. */
  readonly unrunnable: string
  /** What an artifact of a file now stands at, in words a person reads. */
  readonly made: Record<Artifact, Record<ArtifactState, string>>
  /** What a run over the address a note points at came to. */
  readonly fetched: Record<ArtifactState, string>
}

/** The words the step that asks for the name typed back is drawn in. */
export interface RetypeWords {
  /** What it does, and what it leaves behind. */
  readonly action: string
  readonly then: string
  /** What stands in the field: the name of the thing, typed back. */
  readonly back: string
}

/** The words the step that confirms is drawn in. */
export interface ConfirmWords {
  /** The answer that changes nothing, and what it leaves. */
  readonly keeps: string
  readonly kept: string
  /** The answer that does it, and what it leaves. */
  readonly action: string
  readonly then: string
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
  readonly ask: string
  readonly copy: string
  /** The two runs a person asks for over the file in front. */
  readonly transcribe: string
  readonly recognise: string
  /** The transcript of the recording in front, put right by a proofreader. */
  readonly proofread: string
  /** What is at the address a link note points at, fetched again. */
  readonly downloadText: string
  /** What a url points at, fetched onto this disk. */
  readonly downloadCopy: string
  /** The transcript of the recording in front, taken away, and the two answers. */
  readonly deleteText: string
  readonly keepsTranscript: string
  readonly deletes: string
  readonly deleted: string
  /** The copy fetched for the url in front, taken off this disk, and its answers. */
  readonly deleteCopy: string
  readonly keepsCopy: string
  readonly deletedCopy: string
  /** The note in front, shown where the vault files it. */
  readonly reveal: string
  /** The preset the note in front is, or the one the deck in front is scheduled by. */
  readonly preset: string
  readonly newNote: string
  /** The two files a card is written in: the deck it is one of, and what cuts it. */
  readonly newDeck: string
  readonly newStencil: string
  /** The note that says how the decks pointing at it are scheduled. */
  readonly newPreset: string
  /** The note that points at a web address, made from the address. */
  readonly importUrl: string
  readonly newPlex: string
  /** The folders and files of the vault, put in front of the person. */
  readonly files: string
  readonly newAgent: string
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
  /** The command over whether a node hangs the parts of its note under it. */
  readonly hanging: string
  /** The command over how many of them stand under a node at once. */
  readonly parts: string
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
  readonly forgetVault: string
  readonly eraseVault: string
  /** The groups the commands are drawn in. */
  readonly overNote: string
  readonly overFile: string
  readonly overWindow: string
  readonly overVault: string
  /** Why nothing can be done to a note: the vault never opened, or none is in front. */
  readonly noVault: string
  readonly noNote: string
  /** The list of commands: the chip beside the field, and what stands in it. */
  readonly command: string
  readonly typeCommand: string
  /** A name asked for, and the one item it offers. */
  readonly naming: string
  readonly typeName: string
  readonly callIt: string
  /**
   * An address asked for: the field, what stands in it, what Enter does, and
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
  readonly kept: string
  /** The two answers over a vault, whose folder is left where it is. */
  readonly keepsVault: string
  readonly forgets: string
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
  /** The seats it can be made in, off the note in front. */
  readonly asChild: string
  readonly asParent: string
  readonly asJump: string
}
