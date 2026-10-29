/**
 * Command and palette types: what a command is and the context it runs over.
 *
 * This file defines types only. The command table is `lib/table.ts`, availability
 * rules are `lib/availability.ts`, display words are `words.ts`, and the services
 * available while a command runs are `model/deps.ts`.
 */
import type { Component } from 'vue'
import type { PaletteKeys } from '@numen/ui'
import type { EditorKind, FilePlacement } from '@/entities/tab'
import type { VaultList } from '@/entities/vault'
import type { NameMatch } from './model/search/search'
import type { StepGroup } from '@/shared/steps'
import type { ConfirmWords, RetypeWords } from './words'

export type { StepGroup, StepRow } from '@/shared/steps'

/** What the commands ask of the application before anything is chosen. */
export interface CommandsDeps {
  /** The names in the vault that match. */
  names(query: string, limit: number): Promise<readonly NameMatch[]>
  /** Every vault the installation holds, and which of them this window shows. */
  vaults(): Promise<VaultList>
}

/** Which group a command is offered in. */
export type CommandGroup = 'note' | 'file' | 'window' | 'settings' | 'vault'

/**
 * The palette's current step: a prompt for one value, or the list of commands.
 * `picking` queries the vault and `vaults` queries the installation; `choosing`
 * offers an in-memory list; `arguments` prompts for one of a command's `prompts`,
 * one step per prompt.
 */
export type Step =
  'commands' | 'naming' | 'picking' | 'vaults' | 'choosing' | 'asking' | 'exactly' | 'arguments'

/**
 * One value a command requires, entered in its own step. A step that is not
 * `isRequired` can be skipped, and the value is then omitted.
 */
export interface CommandPrompt {
  readonly name: string
  readonly title: string
  readonly isRequired: boolean
}

/** The input a command requires before it runs; each one adds a prompt step. */
export type PromptStep = Exclude<Step, 'commands'>

/** The vault a command is over: the identity the list gives it, and its name. */
export interface VaultRef {
  readonly id: string
  readonly name: string
}

/**
 * The current context: the active tab and the note it resolves to. For an agent
 * tab, the note the plex is focused on.
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
   * The file a run targets. A tab holding a file names that file; a tree row
   * names the file the row represents.
   */
  readonly file: string
  /** The other files it is over, beside the one at `path`. */
  readonly others?: readonly string[]
  readonly vault: VaultRef
  /**
   * Whether the vault accepts operations. A vault still being scanned does: the
   * scan runs in the background and the files it has reached are available.
   */
  readonly isReady: boolean
}

/** A command a user can invoke. */
export interface Command {
  readonly id: string
  readonly text: string
  /** A second line under the command text, such as the plugin that carries it out. */
  readonly detail?: string
  /** The keyboard shortcut that triggers it outside the palette. */
  readonly keys?: PaletteKeys
  /** The icon it is drawn with, where the one who declares it gives one. */
  readonly icon?: Component
  /** The input it requires before it runs. */
  readonly needs?: PromptStep
  /** The step that follows the first one. */
  readonly next?: PromptStep
  /** The values the `arguments` steps ask for, in order. */
  readonly prompts?: readonly CommandPrompt[]
  /** The group it is offered in. */
  readonly group: CommandGroup
  /** Whether it is offered at all over what is in front, in this window. */
  isAvailable(target: CommandTarget, runs: RunSupport): boolean
  getFieldText?(target: CommandTarget): string
  /** What its step displays, when that step confirms or prompts for the name again. */
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
  /** The tab kind the command was invoked from; determines where a note it creates opens. */
  readonly kind: string | null
  /** The tab the command was invoked from, for a command about the tab itself. */
  readonly tab: string
  /** What the `arguments` steps were answered, by the name of each prompt. */
  readonly values?: Readonly<Record<string, string>>
}

/**
 * The lists the window holds, and what it does with the entry under the keyboard
 * selection. A list is re-read every time the step is drawn, so its contents
 * may change while the step is open.
 */
export interface PaletteLists {
  /**
   * What this command offers now, in the groups it is drawn in. The words typed
   * come too: a list may hold a row made out of them.
   */
  getStepGroups(command: string, typed: string): readonly StepGroup[]
  /** The one the keyboard is standing on, and nothing where it relies on none. */
  previewItem(command: string, item: string): void
}

/**
 * What the window records about a note by its file name. The step stays open
 * while the vault changes, so this is re-read each time the step is drawn and
 * once more when the command is invoked.
 */
export interface NoteLookup {
  /** What it is called now, and nothing where the window names it nothing. */
  getTitle(path: string): string
  /** The identity of the tab holding it, and nothing where none holds it. */
  getTabAt(path: string): string | null
}

/**
 * The runs this build cannot perform. The application reports them the first
 * time one is requested; this window omits it from then on.
 */
export interface RunSupport {
  /** Whether this build can do a run at all. A view drawing it follows the answer. */
  canRun(run: string): boolean
  /** A run the application answered it cannot do at all. */
  cannotRun(run: string): void
}

/**
 * The files the window has an editor open on, as accessed by commands. Notes
 * of every type are included.
 */
export interface Notes {
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
  getPath(id: string): string
  asking(id: string): boolean
  /** Answers once nothing of that note is on its way to the file. */
  flushPending(id: string): Promise<void>
  /** The tab holding a note lets go of it. */
  close(id: string): void
  /**
   * Opens a file in the editor matching its type, in its own tab or beside the
   * current one.
   */
  openFile(path: string, title: string, showing: 'current' | 'split'): void
  /** Opens a file created by this command, as the type it was created as. */
  openNewFile(
    path: string,
    title: string,
    options: EditorKind | { type: EditorKind; where?: FilePlacement },
  ): void
}

/**
 * One store of open files, as accessed by commands. Each note-type editor
 * keeps one.
 */
export interface Store {
  /** Whether this store holds a file open under that identity. */
  has(id: string): boolean
  getPath(id: string): string
  /** What it is called, under the identity it opened under. */
  getTitle(id: string): string
  asking(id: string): boolean
  /** Answers once nothing of it is on its way to the file. */
  flushPending(id: string): Promise<void>
  /** The tab holding it lets go of it. */
  close(id: string): void
  /** The identity of the tab standing at a file, and nothing where none does. */
  getTabAt(path: string): string | null
}
