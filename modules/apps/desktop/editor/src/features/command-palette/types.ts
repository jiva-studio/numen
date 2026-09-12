/**
 * What a command is, and what it is asked over.
 *
 * Nothing here decides anything: the table of commands is `lib/table.ts`, when
 * each is offered is `lib/offered.ts`, the words they are drawn in are `words.ts`,
 * and what the window offers one being carried out is `model/deps.ts`.
 */
import type { PaletteKeys } from '@numen/ui'
import type { ArtifactStates } from '@/shared/artifacts'
import type { EditorKind } from '@/entities/tab'
import type { Source } from '@/shared/file'
import type { VaultList } from '@/shared/vaults'
import type { NameMatch } from './model/search'
import type { ConfirmWords, RetypeWords } from './words'

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
  isOffered(at: CommandTarget, runs: RunSupport): boolean
  /** What stands in the field when its step opens, for the person to replace. */
  getFieldText?(at: CommandTarget): string
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

/** One of a list the window itself holds, as the step that offers it draws it. */
export interface StepRow {
  readonly id: string
  readonly title: string
  /** A second line: what is true of this row and not of the ones beside it. */
  readonly detail?: string
  /** Drawn, said, and not chosen. */
  readonly disabled?: boolean
  /** The value the setting this list is of holds now, which is where it opens. */
  readonly inForce?: boolean
}

/** One group of such a list, named by whatever holds it. */
export interface StepGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly StepRow[]
  /** What is said in its place where it holds nothing. */
  readonly silence?: string
}

/**
 * The lists the window holds, and what it does with the one the keyboard is
 * standing on. A list is read again every time the step is drawn, so what the
 * window holds may change while the step stands open.
 */
export interface PaletteLists {
  /**
   * What this command offers now, in the groups it is drawn in. The words typed
   * come too: a list may hold a row made out of them.
   */
  getStepGroups(command: string, typed: string): readonly StepGroup[]
  /** The one the keyboard is standing on, and nothing where it stands on none. */
  previewItem(command: string, item: string): void
}

/**
 * What the window knows about a note by the name it is filed under. A step
 * stands open while the vault moves under it, and this is read again each time
 * the step is drawn and once more as the invocation is made.
 */
export interface NoteLookup {
  /** What it is called now, and nothing where the window names it nothing. */
  getTitle(path: string): string
  /** The identity of the tab holding it, and nothing where none holds it. */
  getTabAt(path: string): string | null
}

/**
 * The runs this build cannot do at all, as one window has been told them. The
 * application says so the first time one is asked for, and that window offers
 * it nowhere after that.
 */
export interface RunSupport {
  /** Whether this build can do a run at all. A view drawing it follows the answer. */
  canRun(run: string): boolean
  /** A run the application answered it cannot do at all. */
  cannotRun(run: string): void
}

/**
 * The files the window has an editor open on, as a command reaches them. A
 * note, a deck and a stencil each answer here.
 */
export interface Notes {
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
  /** The file a note stands at now, under the identity it opened under. */
  getPath(id: string): string
  /** Whether the note owes the person an answer about what its file now holds. */
  isAsking(id: string): boolean
  /** Answers once nothing of that note is on its way to the file. */
  settle(id: string): Promise<void>
  /** The tab holding a note lets go of it. */
  close(id: string): void
  /**
   * A file put in front of the person, in the editor made for what it is, in a
   * tab of its own or one beside it.
   */
  openFile(path: string, title: string, showing: 'here' | 'beside'): void
  /** A file just made here, put in front of the person as what it was made as. */
  openNewFile(path: string, title: string, type: EditorKind, showing: 'here' | 'beside'): void
}

/**
 * One store of open files, as a command reaches what it holds. The notes, the
 * decks and the stencils each keep one.
 */
export interface Store {
  /** Whether this store holds a file open under that identity. */
  has(id: string): boolean
  /** The file one of them stands at now, under the identity it opened under. */
  getPath(id: string): string
  /** What it is called, under the identity it opened under. */
  getTitle(id: string): string
  /** Whether it owes the person an answer about what its file now holds. */
  isAsking(id: string): boolean
  /** Answers once nothing of it is on its way to the file. */
  settle(id: string): Promise<void>
  /** The tab holding it lets go of it. */
  close(id: string): void
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
}
