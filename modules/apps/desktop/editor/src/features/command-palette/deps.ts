/**
 * What the window offers a command being carried out: one port to a job, and
 * the words it answers in.
 *
 * Nothing here does anything. `handlers.ts` is what carries a command out
 * through these, and `vaults.ts` what it does to the vaults.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { RunSupport } from './runs'
import type { CommandInvocation, VaultRef } from './target'
import type { EditorKind } from '@/entities/tab'
import type { Artifact, ArtifactState, ArtifactRunner } from '@/shared/artifacts'
import type { Movement } from '@/shared/file'
import type { RemoveResult, RenameResult } from '@/entities/note'
import type { ErrorCode } from '@/shared/errors'
import type { VaultErrorCode, Vaults } from '@/shared/vaults'
import type { MessageWriter } from '@/shared/notices/messages'


/**
 * The files the window has an editor open on, as a command reaches them. A
 * note, a deck and a stencil each answer here.
 */
export interface Notes {
  /** The identity of the tab standing at a file, and nothing where none does. */
  holding(path: string): string | null
  /** The file a note stands at now, under the identity it opened under. */
  where(id: string): string
  /** Whether the note owes the person an answer about what its file now holds. */
  asking(id: string): boolean
  /** Answers once nothing of that note is on its way to the file. */
  settles(id: string): Promise<void>
  /** The tab holding a note lets go of it. */
  shuts(id: string): void
  /**
   * A file put in front of the person, in the editor made for what it is, in a
   * tab of its own or one beside it.
   */
  opens(path: string, title: string, showing: 'here' | 'beside'): void
  /** A file just made here, put in front of the person as what it was made as. */
  made(path: string, title: string, type: EditorKind, showing: 'here' | 'beside'): void
}

/**
 * One store of open files, as a command reaches what it holds. The notes, the
 * decks and the stencils each keep one.
 */
export interface Store {
  /** Whether this store holds a file open under that identity. */
  has(id: string): boolean
  /** The file one of them stands at now, under the identity it opened under. */
  where(id: string): string
  /** What it is called, under the identity it opened under. */
  called(id: string): string
  /** Whether it owes the person an answer about what its file now holds. */
  asking(id: string): boolean
  /** Answers once nothing of it is on its way to the file. */
  settles(id: string): Promise<void>
  /** The tab holding it lets go of it. */
  shuts(id: string): void
  /** The identity of the tab standing at a file, and nothing where none does. */
  holding(path: string): string | null
}

/**
 * The open files a command reaches, over every store the window keeps them in.
 * An identity is answered by the store holding it, and one nobody holds by
 * nothing at all.
 */
export const reaching = (
  stores: readonly Store[],
  puts: Pick<Notes, 'opens' | 'made'>,
): Notes => {
  const holder = (id: string): Store | undefined => stores.find((one) => one.has(id))
  return {
    holding: (path) => {
      for (const one of stores) {
        const held = one.holding(path)
        if (held !== null) return held
      }
      return null
    },
    where: (id) => holder(id)?.where(id) ?? id,
    asking: (id) => holder(id)?.asking(id) ?? false,
    settles: async (id) => {
      await holder(id)?.settles(id)
    },
    shuts: (id) => holder(id)?.shuts(id),
    opens: puts.opens,
    made: puts.made,
  }
}

/** A note a command made: where it is filed, and what it is called. */
export interface NewNote {
  readonly path: string
  readonly title: string
}

/** The vault as a command changes what it holds. */
export interface VaultWriter {
  /** A note made under the name it is given, in a seat of another one. */
  makes(title: string, from: string, seat: PlexRelatedSeat | null): Promise<NewNote | null>
  /** A note given a different name, and its file renamed with it where the two are one name. */
  renames(path: string, title: string): Promise<RenameResult>
  /** A note taken out of the vault, into the trash or off the disk. */
  removes(path: string, destroy: boolean): Promise<RemoveResult>
  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what
   * it is called from now on.
   */
  moves(from: string, to: string): Promise<Movement>
  /** An empty folder. The folders above it are made with it. */
  makesFolder(path: string): Promise<ErrorCode | null>
}

/** The files a command makes from nothing, each put in front of the person. */
export interface FileMakers {
  /**
   * A deck made in a folder under the name it is given. The path it landed at,
   * and nothing where none was made.
   */
  decks(folder: string, name: string): Promise<string>
  /** A stencil made the same way. */
  stencils(folder: string, name: string): Promise<string>
  /** A preset made the same way, naming none of its settings. */
  presets(folder: string, name: string): Promise<string>
  /**
   * A note pointing at an address, named by the address until what is at it
   * says what it is called. The path it landed at, and nothing where none was
   * made.
   */
  imports(folder: string, address: string): Promise<string>
}

/** The vaults this installation holds, as a command changes which one shows. */
export interface VaultSwitcher extends Vaults {
  /** The vault the window is showing, under the name it has now. */
  calls(vault: VaultRef): void
  /**
   * The page drawn again, on the vault the window shows now. Every tab and
   * every plex belonged to the vault that has gone.
   */
  reloads(): void
}

/** Where a command takes the window. */
export interface WindowNavigator {
  /** The files of the vault put in front of the person, opened down to a path. */
  reveals(path: string): void
  /** A note put in front of the person, in the plex they are looking at. */
  travel(path: string): Promise<void>
  /** Every plex standing on a note travels to another one. */
  leaves(from: string, to: string): Promise<void>
  /** The note the vault opens with. */
  opening(): string
  /** A tab of a kind, opened and put in front. */
  opens(kind: string): void
  /**
   * The preset of a note, in a tab of its own: the note itself where it is one,
   * and the preset a deck is scheduled by where it is a deck.
   */
  preset(path: string): Promise<void>
  /** A tab let go of. */
  closes(tab: string): void
  /** Something to ask, put in the agent the person was last in. */
  asks(text: string): void
  /** The search, in place of the commands. */
  searches(): void
}

/** The settings a command writes, each under the identity the window gives it. */
export interface SettingsWriter {
  /**
   * The window drawn another way: a theme worn from now on, which half of a
   * colour pair the tokens are read as, or how large one of the two kinds of
   * text is set.
   */
  appearance(chosen: string): Promise<void>
  /** Whether a note's title and the name of its file are kept as one name. */
  syncing(chosen: string): Promise<void>
  /** Whether a node hangs the parts of its note under it. */
  hanging(chosen: string): Promise<void>
  /** How many parts a node hangs at once. */
  parts(chosen: string): Promise<void>
}

/** What the window offers a command being carried out, one port to a job. */
export interface CommandDeps {
  readonly files: VaultWriter
  readonly runs: ArtifactRunner
  readonly makers: FileMakers
  readonly vaults: VaultSwitcher
  readonly goes: WindowNavigator
  readonly settings: SettingsWriter
  /** The open files a command reaches, whichever store holds each. */
  readonly notes: Notes
  /** The runs this window has been told this build cannot do. */
  readonly runSupport: RunSupport
  /** A path put on the clipboard. */
  copies(path: string): void
  /**
   * What was done, or could not be, in words a person reads. One command's
   * word replaces the last, and nothing said clears it.
   */
  readonly says: MessageWriter
}

/** Everything carrying a command out says in the window's voice. */
export interface Words {
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

/** One command, carried out. */
export type CommandHandler = (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: Words,
) => Promise<void> | void

/** What a command did to other notes, named once each under what it did. */
export const naming = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Everything one command has to say, as the one line the window says it in. */
export const all = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
