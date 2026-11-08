/**
 * Command and palette types: what a command is and the context it runs over.
 *
 * This file defines types only. The command table is `lib/table.ts`, availability
 * rules are `lib/availability.ts`, display words are `words.ts`, and the services
 * available while a command runs are `model/deps.ts`.
 */
import type { Component } from 'vue'
import type { Selector } from '@numen/plugin-host'
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
 * Each kind is a noun for what the person is asked: `name` is typed, `note` queries the
 * vault, `vault` queries the installation, `choice` offers an in-memory list,
 * `confirmation` asks for a yes or a no, `retypedName` asks for the title typed again, and
 * `arguments` prompts for one of a command's `prompts`, one step per prompt.
 */
export type Step =
  'commands' | 'name' | 'note' | 'vault' | 'choice' | 'confirmation' | 'retypedName' | 'arguments'

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

/** Where in the file a command that opens it moves to: a line of a note, or the selector of a reader. */
export interface CommandPlace {
  readonly line?: number
  readonly selector?: Selector
}

/**
 * The current context: the active tab and the note it resolves to. For an agent
 * tab, the note the plex is focused on.
 */
export interface CommandTarget {
  readonly tab: string
  readonly kind: string | null
  readonly path: string
  readonly title: string
  readonly file: string
  readonly line?: number
  readonly selector?: Selector
  readonly others?: readonly string[]
  readonly vault: VaultRef
  readonly isReady: boolean
}

/** A command a user can invoke. */
export interface Command {
  readonly id: string
  readonly text: string
  readonly detail?: string
  readonly keys?: PaletteKeys
  readonly icon?: Component
  readonly iconName?: string
  readonly input?: PromptStep
  readonly next?: PromptStep
  readonly prompts?: readonly CommandPrompt[]
  readonly group: CommandGroup
  /** Whether it is offered at all over what is in front, in this window. */
  isAvailable(target: CommandTarget, runs: RunSupport): boolean
  getFieldText?(target: CommandTarget): string
  readonly answers?: ConfirmWords
  readonly warns?: RetypeWords
  readonly also?: string
}

/** One command as it is carried out: what it is over, and what was typed for it. */
export interface CommandInvocation {
  readonly id: string
  readonly path: string
  readonly vault: VaultRef
  readonly note: string | null
  readonly title: string
  readonly file: string
  readonly line?: number
  readonly selector?: Selector
  readonly others: readonly string[]
  readonly name: string
  readonly kind: string | null
  readonly tab: string
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
  openFile(path: string, title: string, showing: 'current' | 'split', at?: CommandPlace): void
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
