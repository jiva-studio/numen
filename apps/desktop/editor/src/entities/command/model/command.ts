/**
 * What a command is: the row a feature declares, the target it is over, and the invocation that
 * carries it out.
 */
import type { Component } from 'vue'
import type { CommandItem, Selector } from '@numen/plugin-loader'
import type { Keystroke } from '@numen/ui'

/** Which group a command is offered in. */
export type CommandGroup = 'note' | 'file' | 'window' | 'settings' | 'vault'

/**
 * The palette's current step: a prompt for one value, or the list of commands.
 * Each kind is a noun for what the person is asked: `name` is typed, `note` queries the
 * vault, `vault` queries the installation, `items` offers an in-memory list,
 * `confirmation` asks for a yes or a no, `retypedName` asks for the title typed again, and
 * `arguments` prompts for one of a command's `prompts`, one step per prompt.
 */
type Step =
  'commands' | 'name' | 'note' | 'vault' | 'items' | 'confirmation' | 'retypedName' | 'arguments'

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

/** Text for the step that requires the user to retype a name. */
export interface RetypeWords {
  readonly action: string
  readonly detail: string
  readonly back: string
}

/** The words the step that confirms is drawn in. */
export interface ConfirmWords {
  readonly keeps: string
  readonly retainedNotice: string
  readonly action: string
  readonly detail: string
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
  readonly isFolder?: boolean
  readonly others?: readonly string[]
  readonly isTargetNamed?: boolean
  readonly vault: VaultRef
  readonly isReady: boolean
}

/**
 * What carries a command out. A command that made a note or a folder answers with its path; one
 * that made none answers nothing.
 */
export type CommandRun = (invocation: CommandInvocation) => Promise<string | void> | string | void

/**
 * What a feature declares: the rows it offers, and what carries out each command by its id. A row
 * that runs another command through `command` has no handler of its own, and a command that is
 * only run, as by a plugin, has a handler and no row.
 */
export interface FeatureCommands {
  readonly commands: readonly Command[]
  readonly handlers: Readonly<Record<string, CommandRun>>
}

/**
 * A command a user can invoke, as one row. A row is run under its `id`; a row whose `command` is
 * set runs that command with its `values` as the arguments.
 */
export interface Command {
  readonly id: string
  readonly command?: string
  readonly values?: Readonly<Record<string, string>>
  readonly text: string
  readonly detail?: string
  readonly keys?: Keystroke
  readonly icon?: Component
  readonly iconName?: string
  readonly input?: PromptStep
  readonly canOmitNameOverFolder?: boolean
  readonly next?: PromptStep
  readonly prompts?: readonly CommandPrompt[]
  readonly group: CommandGroup
  readonly when?: string
  /** Whether it is offered at all over what is in front, in this window. */
  isAvailable(target: CommandTarget, runs: RunSupport): boolean
  getFieldText?(target: CommandTarget): string
  readonly answers?: ConfirmWords
  readonly warns?: RetypeWords
  readonly also?: string
  /**
   * The items it offers over the target, for a command whose step is a argument; the argument picked
   * is the name of its invocation.
   */
  listItems?(target: CommandTarget): readonly CommandItem[] | Promise<readonly CommandItem[]>
}

/** What a command was invoked over: the target of the palette without whether it is ready. */
export interface InvocationTarget {
  readonly path: string
  readonly vault: VaultRef
  readonly title: string
  readonly file: string
  readonly line?: number
  readonly selector?: Selector
  readonly isFolder?: boolean
  readonly isTargetNamed?: boolean
  readonly others: readonly string[]
  readonly kind: string | null
  readonly tab: string
}

/**
 * One command as it is carried out: what it is over, the tab the note is open in, and what was
 * typed for it (the name) or set for it (the values).
 */
export interface CommandInvocation {
  readonly id: string
  readonly target: InvocationTarget
  readonly note: string | null
  readonly name: string
  readonly values?: Readonly<Record<string, string>>
}
