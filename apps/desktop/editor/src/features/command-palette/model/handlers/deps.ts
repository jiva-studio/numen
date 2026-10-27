/**
 * What the window offers a command being carried out, one port to a job.
 *
 * Nothing here does anything. `handlers.ts` is what carries a command out
 * through these, `vaults.ts` what it does to the vaults, and `../lib/voice.ts`
 * how what it has to say becomes one line.
 */
import type { Movement } from '@/entities/file'
import type { NoteRelation, RemoveResult, RenameResult } from '@/entities/note'
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
  createNote(title: string, from: string, relation: NoteRelation | null): Promise<NewNote | null>
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

/** The files a command makes from nothing, each put in front of the person. */
export interface FileMakers {
  /**
   * A note pointing at an address, named by the address until what is at it
   * says what it is called. The path it landed at, and nothing where none was
   * made.
   */
  createUrl(folder: string, address: string): Promise<string>
}

/** The vaults this installation holds, as a command changes which one shows. */
export interface VaultSwitcher extends Vaults {
  /** One vault of the list under the name it was just given. */
  setVaultName(vault: VaultRef): void
  /**
   * The page drawn again, on the vault the window shows now. Every tab and
   * every plex belonged to the vault that has gone.
   */
  reload(): void
}

/** Where a command takes the window. */
export interface WindowNavigator {
  /** A note put in front of the person, where the plugin that shows notes puts it. */
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
   * The window drawn another way: a theme worn from now on, which half of a
   * colour pair the tokens are read as, or how large one of the two kinds of
   * text is set.
   */
  chooseAppearance(selected: string): Promise<void>
  /** Whether a note's title and the name of its file are kept as one name. */
  chooseSync(selected: string): Promise<void>
  /** The performance workload profile chosen. */
  choosePerformance(selected: string): Promise<void>
  /** A declarative plugin setting chosen. */
  choosePluginSetting?(pluginId: string, key: string, selected: unknown): Promise<void>
}

/** The one line the window says a command's answer in. */
export interface Voice {
  /**
   * What was done, or could not be, in words a person reads. One command's
   * word replaces the last, and nothing said clears it.
   */
  readonly writeMessage: MessageWriter
}

/** The vault a command reaches: what it writes, what it makes, and which one shows. */
export interface VaultContext {
  readonly files: VaultWriter
  readonly makers: FileMakers
  readonly vaults: VaultSwitcher
}

/** The tabs a command reaches: the files they hold, and where the window goes. */
export interface TabContext {
  /** The open files a command reaches, whichever store holds each. */
  readonly notes: Notes
  readonly navigator: WindowNavigator
}

/** The runs this build has said it cannot do. */
export interface RunContext {
  /** The runs this window has been told this build cannot do. */
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
