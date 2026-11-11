/**
 * What the window offers a command being carried out, one port to a job.
 *
 * Nothing here does anything. `handlers.ts` is what carries a command out through these.
 */
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { CommandInvocation, RunSupport } from './invocation'
import type { Notes } from './notes'
import type { CommandRegistry } from './registry'

/** Where a command of the palette takes the window. */
interface WindowNavigator {
  /** A tab of a kind, opened and put in front. */
  openTab(kind: string): void
  /** A tab release. */
  closeTab(tab: string): void
  /** The search, in place of the commands. */
  search(): void
}

/** The settings a command writes. */
interface SettingsWriter {
  /**
   * The row chosen in the step of the command of a setting, `setting:<owner>:<key>`; the adapter
   * of the settings writes it.
   */
  chooseSetting(command: string, selected: string): Promise<void>
}

/** Reports a command's result as one line of text. */
interface Voice {
  readonly writeMessage: MessageWriter
}

/** The tabs a command operates on: their files, and where the window navigates. */
interface TabContext {
  readonly notes: Notes
  readonly navigator: WindowNavigator
}

/** The runs this build does not support. */
interface RunContext {
  readonly runSupport: RunSupport
}

/** What the window offers a command being carried out, one port to a job. */
export interface CommandDeps extends TabContext, RunContext, Voice {
  readonly registry: Pick<CommandRegistry, 'get'>
  readonly settings: SettingsWriter
  readonly emitEvent?: ((event: string) => void) | undefined
}

/** One command, carried out. A command that made a note or a folder answers with its path. */
export type CommandHandler = (
  invocation: CommandInvocation,
  on: CommandDeps,
) => Promise<string | void> | string | void
