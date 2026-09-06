/**
 * Carrying out a command, over what the window offers to do it with.
 *
 * Every command is one entry here, under the identity `commands.ts` gives
 * it, so a command that is offered and a command that happens are the same
 * list. Nothing here draws anything.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import { troubleWords } from '@numen/wire'
import type { CommandInvocation, RunSupport, VaultRef } from './commands'
import type { EditorKind } from '../tabs/openers'
import type {
  Artifact,
  Movement,
  Outcome,
  ArtifactState,
  RefusalReason,
  RemoveResult,
  RenameResult,
  ArtifactRunner,
  VaultRefusalReason,
  Vaults,
} from '../core'
import type { MessageWriter } from '../notices/messages'
import { AGENT, FILES, NOTE, PLEX, SETTINGS } from '../tabs/workspace'

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
  makesFolder(path: string): Promise<RefusalReason | null>
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
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<RefusalReason, string>
  /** What the list of vaults refused, in words a person reads. */
  readonly unvaulted: Record<VaultRefusalReason, string>
  /** What the machine's own folder picker is titled. */
  readonly folder: string
  /** The links that reach nothing now, which nothing repairs. */
  readonly dangling: string
  /** The vault opens with no note at all. */
  readonly nowhere: string
  /** The note is waiting on the person, and its file stays where it is. */
  readonly unanswered: string
  /** The note holds prose nobody here has seen, so nothing was written. */
  readonly overtaken: string
  /** A name at the destination is taken, and the file stayed where it was. */
  readonly occupied: string
  /** This build cannot do the run at all, and stops offering it. */
  readonly unrunnable: string
  /** What an artifact of a file now stands at, in words a person reads. */
  readonly made: Record<Artifact, Record<ArtifactState, string>>
}

/** One command, carried out. */
type CommandHandler = (
  invocation: CommandInvocation,
  on: CommandDeps,
  words: Words,
) => Promise<void> | void

/** What each command comes to. A command with no entry here does nothing. */
const carried: Record<string, CommandHandler> = {
  read: (invocation, on) => on.notes.opens(invocation.path, invocation.title, 'here'),
  beside: (invocation, on) => on.notes.opens(invocation.path, invocation.title, 'beside'),
  travel: (invocation, on) => on.goes.travel(invocation.path),
  child: (invocation, on, words) => makes(invocation, 'child', on, words),
  parent: (invocation, on, words) => makes(invocation, 'parent', on, words),
  jump: (invocation, on, words) => makes(invocation, 'jump', on, words),
  note: (invocation, on, words) => makes(invocation, null, on, words),
  deck: async (invocation, on) => {
    if (invocation.name) await on.makers.decks('', invocation.name)
  },
  stencil: async (invocation, on) => {
    if (invocation.name) await on.makers.stencils('', invocation.name)
  },
  newPreset: async (invocation, on) => {
    if (invocation.name) await on.makers.presets('', invocation.name)
  },
  importUrl: async (invocation, on) => {
    if (invocation.name) await on.makers.imports('', invocation.name)
  },
  title: (invocation, on, words) => renames(invocation, on, words),
  remove: (invocation, on, words) => removes(invocation, false, on, words),
  destroy: (invocation, on, words) => removes(invocation, true, on, words),
  transcribe: async (invocation, on, words) =>
    began(invocation, await on.runs.makes(invocation.file, 'transcript'), on, words),
  recognise: async (invocation, on, words) =>
    began(invocation, await on.runs.makes(invocation.file, 'reading'), on, words),
  proofread: async (invocation, on, words) =>
    began(invocation, await on.runs.makes(invocation.file, 'corrections'), on, words),
  dropTranscript: async (invocation, on, words) => {
    if (await on.runs.drops(invocation.file)) return
    on.runSupport.cannotRun(invocation.id)
    on.says(words.unrunnable, 'refusal')
  },
  ask: (invocation, on) => on.goes.asks(`${invocation.path} — `),
  copy: (invocation, on) => on.copies(invocation.path),
  reveal: (invocation, on) => on.goes.reveals(invocation.path),
  preset: (invocation, on) => on.goes.preset(invocation.path),
  settings: (_, on) => on.goes.opens(SETTINGS),
  move: (invocation, on, words) => moves(invocation, on, words),
  makeFolder: (invocation, on, words) => makesFolder(invocation, on, words),
  plex: (_, on) => on.goes.opens(PLEX),
  files: (_, on) => on.goes.opens(FILES),
  agent: (_, on) => on.goes.opens(AGENT),
  close: (invocation, on) => on.goes.closes(invocation.tab),
  find: (_, on) => on.goes.searches(),
  appearance: (invocation, on) => on.settings.appearance(invocation.name),
  mode: (invocation, on) => on.settings.appearance(invocation.name),
  interfaceScale: (invocation, on) => on.settings.appearance(invocation.name),
  textScale: (invocation, on) => on.settings.appearance(invocation.name),
  syncing: (invocation, on) => on.settings.syncing(invocation.name),
  hanging: (invocation, on) => on.settings.hanging(invocation.name),
  parts: (invocation, on) => on.settings.parts(invocation.name),
  first: (_, on, words) => travels(on.goes.opening(), on, words),
  goto: (invocation, on, words) => travels(invocation.path, on, words),
  openVault: (invocation, on, words) => shows(invocation.vault.id, on, words),
  newVault: (_, on, words) => adds(on, words),
  renameVault: (invocation, on, words) => calls(invocation, on, words),
  forgetVault: (invocation, on, words) => forgets(invocation, false, on, words),
  eraseVault: (invocation, on, words) => forgets(invocation, true, on, words),
}

/** A command carried out. Nothing chosen does nothing at all. */
export async function does(invocation: CommandInvocation | null, on: CommandDeps, words: Words): Promise<void> {
  if (!invocation) return
  const carry = carried[invocation.id]
  if (!carry) return
  on.says('')
  try {
    await carry(atItsFile(invocation, on), on, words)
  } catch (error) {
    on.says(troubleWords(error), 'refusal')
  }
}

/**
 * The invocation at the file its note stands at now. One over a note no tab of the
 * window holds is at the name it was made over.
 */
const atItsFile = (invocation: CommandInvocation, on: CommandDeps): CommandInvocation =>
  invocation.note ? { ...invocation, path: on.notes.where(invocation.note) } : invocation

/** A tab asked to settle: which one it was, and whether it is still waiting. */
interface SettleResult {
  readonly held: string | null
  readonly waiting: boolean
}

/**
 * The tab holding a note, once nothing of the note is on its way to its file.
 * A tab waiting on the person to answer for it settles nothing and says so.
 */
const settles = async (path: string, on: CommandDeps): Promise<SettleResult> => {
  const held = on.notes.holding(path)
  if (held === null) return { held, waiting: false }
  if (on.notes.asking(held)) return { held, waiting: true }
  await on.notes.settles(held)
  return { held, waiting: false }
}

/**
 * A note made under the name that was typed. From a note tab it opens in a tab
 * beside; anywhere else the plex the person is looking at travels to it.
 */
const makes = async (
  invocation: CommandInvocation,
  seat: PlexRelatedSeat | null,
  on: CommandDeps,
  words: Words,
): Promise<void> => {
  if (!invocation.name) return
  const made = await on.files.makes(invocation.name, seat ? invocation.path : '', seat)
  if (!made) return
  if (invocation.kind === NOTE) return on.notes.made(made.path, made.title, 'note', 'beside')
  await travels(made.path, on, words)
}

/**
 * A note given a different name, and its file renamed with it where the two are
 * one name. Prose on disk that nobody here has seen leaves the note as it is.
 */
const renames = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.title) return
  const tab = await settles(invocation.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.renames(invocation.path, invocation.name)
  if (answer.changed) return on.says(words.overtaken, 'caution')
  if (answer.refusal) on.says(words.refused[answer.refusal], 'refusal')
}

/**
 * A file or a folder filed somewhere else, carrying the name the path ends in.
 * A destination that is taken leaves it where it was.
 */
const moves = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.path) return
  const tab = await settles(invocation.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.moves(invocation.path, invocation.name)
  if (answer.refusal === 'occupied') return on.says(words.occupied, 'refusal')
  if (answer.refusal) on.says(words.refused[answer.refusal], 'refusal')
}

/** What an artifact stands at while a run is under way, which is said as a report. */
const UNDER_WAY: readonly ArtifactState[] = ['queued', 'running']

/**
 * An artifact asked for over a file. What it now stands at is one troubleWords,
 * which is what the person is told; a run under way shows what it is doing in
 * the work behind the window. A build that cannot make it at all is told once
 * and offers it nowhere after that.
 */
const began = (invocation: CommandInvocation, outcome: Outcome, on: CommandDeps, words: Words): void => {
  if (!outcome.able) {
    on.runSupport.cannotRun(invocation.id)
    return on.says(words.unrunnable, 'refusal')
  }
  // A file nothing here could read carries what the run said about it, and that
  // stands after the troubleWords.
  const why = words.made[outcome.of][outcome.made]
  const said = outcome.error ? `${why} ${outcome.error}` : why
  on.says(said, UNDER_WAY.includes(outcome.made) ? 'report' : 'refusal')
}

/** An empty folder, made under the path that was typed. */
const makesFolder = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name) return
  const refusal = await on.files.makesFolder(invocation.name)
  if (refusal === 'occupied') return on.says(words.occupied, 'refusal')
  if (refusal) on.says(words.refused[refusal], 'refusal')
}

/**
 * The files one invocation is over: the one it names, and the rest of the selection it
 * was asked over.
 */
const over = (invocation: CommandInvocation): readonly string[] => [invocation.path, ...invocation.others]

/**
 * Files taken out of the vault. A file that has gone is gone from the tree, so
 * the notes left pointing at nothing are the whole of what is said. One the
 * vault refuses leaves the rest to go, and what was refused is what the person
 * is told.
 */
const removes = async (invocation: CommandInvocation, destroy: boolean, on: CommandDeps, words: Words): Promise<void> => {
  const dangling: string[] = []
  const refused: string[] = []
  let waiting = false
  const opening = on.goes.opening()

  for (const path of over(invocation)) {
    const tab = await settles(path, on)
    if (tab.waiting) {
      waiting = true
      continue
    }
    const answer = await on.files.removes(path, destroy)
    if (answer.refusal) {
      refused.push(words.refused[answer.refusal])
      continue
    }
    if (tab.held) on.notes.shuts(tab.held)
    dangling.push(...answer.dangling)
    if (opening) await on.goes.leaves(path, opening)
  }

  if (refused.length > 0) return on.says(all(...refused), 'refusal')
  if (waiting) return on.says(words.unanswered, 'caution')
  on.says(naming(words.dangling, dangling))
}

/**
 * Another vault under this window. What the page holds belongs to the vault
 * that has gone, so the page is drawn again on the one that arrived.
 */
const shows = async (id: string, on: CommandDeps, words: Words): Promise<void> => {
  if (!id) return
  const refusal = await on.vaults.open(id)
  if (refusal) return on.says(words.unvaulted[refusal], 'refusal')
  on.vaults.reloads()
}

/**
 * A folder chosen on this machine, added as a vault and opened. A person who
 * chose no folder has asked for nothing.
 */
const adds = async (on: CommandDeps, words: Words): Promise<void> => {
  const path = await on.vaults.choose(words.folder)
  if (!path) return
  const answer = await on.vaults.add(path, '')
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (!answer.vault) return
  await shows(answer.vault.name, on, words)
}

/** A vault called something else. Its folder keeps the name it has on disk. */
const calls = async (invocation: CommandInvocation, on: CommandDeps, words: Words): Promise<void> => {
  if (!invocation.name || invocation.name === invocation.vault.name) return
  const answer = await on.vaults.rename(invocation.vault.id, invocation.name)
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (answer.vault) on.vaults.calls({ id: answer.vault.name, name: answer.vault.displayName })
}

/**
 * A vault taken off the list. Erasing it puts the folder in the trash this
 * machine keeps; forgetting it leaves the folder where it is.
 */
const forgets = async (invocation: CommandInvocation, erase: boolean, on: CommandDeps, words: Words): Promise<void> => {
  const id = invocation.vault.id
  const refusal = await on.vaults.remove(id, erase)
  if (refusal) on.says(words.unvaulted[refusal], 'refusal')
}

/** A note travelled to, and a vault with none to travel to said. */
const travels = async (path: string, on: CommandDeps, words: Words): Promise<void> => {
  if (!path) return on.says(words.nowhere, 'caution')
  await on.goes.travel(path)
}

/** What a command did to other notes, named once each under what it did. */
const naming = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Everything one command has to say, as the one line the window says it in. */
const all = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
