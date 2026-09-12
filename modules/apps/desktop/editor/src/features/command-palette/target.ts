/**
 * What a command is, and what it is asked over.
 *
 * Nothing here decides anything: the table of commands is `table.ts`, when each
 * is offered is `where.ts`, and the words they are drawn in are `words.ts`.
 */
import type { PaletteKeys } from '@numen/ui'
import type { ArtifactStates } from '@/shared/artifacts'
import type { Source } from '@/shared/file'
import type { VaultList } from '@/shared/vaults'
import type { RunSupport } from './runs'
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
export const isCommandsTyped = (was: string, now: string): boolean => was === '' && now === '>'
