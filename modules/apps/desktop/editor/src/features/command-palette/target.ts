/**
 * What a command is, what it is asked over, and the words it is offered in.
 *
 * Nothing here decides anything: the table of commands is `table.ts` and
 * when each is offered is `where.ts`. This is what both of them speak.
 */
import type { PaletteKeys } from '@numen/ui'
import type { ArtifactStates } from '@/shared/artifacts'
import type { Source } from '@/shared/file'
import type { VaultList } from '@/shared/vaults'
import type { RunSupport } from './runs'
import type { EmptyWords, NameMatch } from './model/search'

/** What the commands ask of the application before anything is chosen. */
export interface CommandsDeps {
  /** The names in the vault that match. */
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  /** Every vault the installation holds, and which of them this window shows. */
  vaults(): Promise<VaultList>
}

/** Which group a command is offered in. */
export type CommandGroup = 'note' | 'file' | 'window' | 'vault'

/**
 * Which step the palette is on: one being asked for, or the list of commands.
 * `picking` asks the vault what it holds and `vaults` asks the installation;
 * `choosing` offers a list the window holds already.
 */
export type Step =
  | 'commands'
  | 'naming'
  | 'address'
  | 'picking'
  | 'vaults'
  | 'choosing'
  | 'asking'
  | 'exactly'

/** What a command wants before it can happen, which is the step that asks. */
export type PromptStep = Exclude<Step, 'commands'>

/** The vault a command is over: the identity the list gives it, and its name. */
export interface VaultRef {
  readonly id: string
  readonly name: string
}

/**
 * What is in front of the person, and the note it means. An agent tab means
 * the note the plex is standing on.
 */
export interface CommandTarget {
  /** The tab in front, for a command about the tab itself. */
  readonly tab: string
  /** The word its kind is filed under, and nothing for a tab holding nothing. */
  readonly kind: string | null
  /** The note it means, and nothing where it means none. */
  readonly path: string
  readonly title: string
  /**
   * The file a run is over, and what the vault holds there. A tab holding a
   * book or a recording names the file it holds; a row of the tree names the
   * file the row stands for.
   */
  readonly file: string
  readonly source: Source | null
  /**
   * What that file carries, and what has become of each. A file nothing has
   * been asked about carries nothing here, and a command over it is offered on
   * its kind alone.
   */
  readonly made: ArtifactStates
  /** The other files it is over, beside the one at `path`. */
  readonly others?: readonly string[]
  /** The vault the window is showing, and nothing where it shows none. */
  readonly vault: VaultRef
  /**
   * Whether the vault can be asked to do anything. A vault still being read
   * can: the walk runs behind the window, and what it has reached already
   * answers.
   */
  readonly ready: boolean
}

/** The words the step that asks for the name typed back is drawn in. */
export interface RetypeWords {
  /** What it does, and what it leaves behind. */
  readonly does: string
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
  readonly does: string
  readonly then: string
}

/** One thing a person can ask for. */
export interface Command {
  readonly id: string
  readonly text: string
  /** The keystroke that reaches it away from the palette. */
  readonly keys?: PaletteKeys
  /** What it asks for before it happens. */
  readonly needs?: PromptStep
  /** The step it asks for once the first one is answered. */
  readonly next?: PromptStep
  /** The group it is offered in. */
  readonly group: CommandGroup
  /** Whether it is offered at all over what is in front, in this window. */
  where(at: CommandTarget, runs: RunSupport): boolean
  /** What stands in the field when its step opens, for the person to replace. */
  filled?(at: CommandTarget): string
  /** What its step says, where that step confirms or asks for the name back. */
  readonly answers?: ConfirmWords
  readonly warns?: RetypeWords
  /** The command Shift and Enter reach on the same row. */
  readonly also?: string
}

/** One command as it is carried out: what it is over, and what was typed for it. */
export interface CommandInvocation {
  readonly id: string
  /** The note it is over. Empty for a command over the window or the vault. */
  readonly path: string
  /** The vault it is over, which is the one the window shows until a step picks another. */
  readonly vault: VaultRef
  /**
   * The identity of the tab holding that note, and nothing where none holds it.
   * A note that moves is at another name by the time the invocation is carried out.
   */
  readonly note: string | null
  readonly title: string
  /** The file a run is over, which is the one `CommandTarget` named. */
  readonly file: string
  /** The other files it is over, beside the one at `path`. */
  readonly others: readonly string[]
  /**
   * What was typed for it: a name to give, or a name typed back. A step that
   * offers a list the window holds puts the one that was chosen here.
   */
  readonly name: string
  /** The kind of tab it was asked from, which is where a note it makes lands. */
  readonly kind: string | null
  /** The tab it was asked from, for a command about the tab itself. */
  readonly tab: string
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


/** One command as it is carried out, over what it was asked over. */
export const invocationOf = (
  id: string,
  at: CommandTarget,
  name = '',
  note: string | null = null,
): CommandInvocation => ({
  id,
  path: at.path,
  vault: at.vault,
  note,
  title: at.title,
  file: at.file,
  others: at.others ?? [],
  name,
  kind: at.kind,
  tab: at.tab,
})

/**
 * Whether what was typed asks for the commands: the field held nothing, and
 * what went into it is the one character that means them.
 */
export const asksCommands = (was: string, now: string): boolean => was === '' && now === '>'
