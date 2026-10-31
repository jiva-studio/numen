/**
 * What the window offers a command being carried out, one port to a job.
 *
 * Nothing here does anything. `handlers.ts` is what carries a command out
 * through these, `vaults.ts` what it does to the vaults, and `../lib/voice.ts`
 * how what it has to say becomes one line.
 */
import type { Movement } from '@/entities/file'
import type { ConnectionRole } from '@numen/plugin-sdk'
import type { RemoveResult, RenameResult } from '@/entities/note'
import type { Vaults } from '@/entities/vault'
import type { ErrorCode } from '@/shared/errors'
import type { MessageWriter } from '@/shared/notices/messages'
import type { CommandInvocation, Notes, RunSupport, VaultRef } from '../../types'
import type { AnswerWords } from '../../words'

/** A note a command made: where it is filed, and what it is called. */
export interface NewNote {
  readonly path: string
  readonly title: string
}

/** The vault as a command changes what it holds. */
export interface VaultWriter {
  /** A note made under the name it is given, in a relation of another one. */
  createNote(title: string, from: string, relation: ConnectionRole | null): Promise<NewNote | null>
  /** A note given a different name, and its file renamed with it where the two are one name. */
  rename(path: string, title: string): Promise<RenameResult>
  /** A note taken out of the vault, into the trash or off the disk. */
  remove(path: string, isPermanent: boolean): Promise<RemoveResult>
  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what
   * it is called from now on.
   */
  move(from: string, to: string): Promise<Movement>
  /** An empty folder. The folders above it are made with it. */
  createFolder(path: string): Promise<ErrorCode | null>
}

/** The vaults this installation holds, as a command changes which one shows. */
export interface VaultSwitcher extends Vaults {
  /** Updates the displayed name of one vault in the list. */
  setVaultName(vault: VaultRef): void
  /**
   * The page drawn again, on the vault the window shows now. Every tab and
   * every plex belonged to the vault that has gone.
   */
  reload(): void
}

/** Where a command takes the window. */
export interface WindowNavigator {
  /** Shows a note in the plugin that displays notes. */
  travel(path: string): Promise<void>
  /** Every tab standing on a note moves to another one. */
  leave(from: string, to: string): Promise<void>
  /** The note the vault opens with. */
  getOpeningNote(): string
  /** A tab of a kind, opened and put in front. */
  openTab(kind: string): void
  /** A tab release. */
  closeTab(tab: string): void
  /** The search, in place of the commands. */
  search(): void
}

/** The settings a command writes, each under the identity the window gives it. */
export interface SettingsWriter {
  /**
   * Applies an appearance setting: the theme, the color scheme the tokens are
   * read as, or the scale of the interface or the text.
   */
  chooseAppearance(selected: string): Promise<void>
  /** Whether a note's title and the name of its file are kept as one name. */
  chooseSync(selected: string): Promise<void>
  /** The performance workload profile chosen. */
  choosePerformance(selected: string): Promise<void>
  /** A declarative plugin setting chosen. */
  choosePluginSetting?(pluginId: string, key: string, selected: unknown): Promise<void>
}

/** Reports a command's result as one line of text. */
export interface Voice {
  /**
   * Message describing the outcome of a command. Each message replaces the
   * previous one and is not cleared automatically.
   */
  readonly writeMessage: MessageWriter
}

/** The vault a command operates on: what it writes, and which one is active. */
export interface VaultContext {
  readonly files: VaultWriter
  readonly vaults: VaultSwitcher
}

/** The tabs a command operates on: their files, and where the window navigates. */
export interface TabContext {
  /** The open files a command operates on, whichever store holds each. */
  readonly notes: Notes
  readonly navigator: WindowNavigator
}

/** The runs this build does not support. */
export interface RunContext {
  /** The runs this build does not support, as recorded by this window. */
  readonly runSupport: RunSupport
}

/** What the window offers a command being carried out, one port to a job. */
export interface CommandDeps extends VaultContext, TabContext, RunContext, Voice {
  readonly settings: SettingsWriter
  /** A path put on the clipboard. */
  copyPath(path: string): void
  readonly emitEvent?: (event: string) => void
}

/** One command, carried out. */
export type CommandHandler = (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: AnswerWords,
) => Promise<void> | void
