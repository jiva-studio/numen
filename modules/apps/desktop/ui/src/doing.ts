/**
 * Carrying out a command, over what the window offers to do it with.
 *
 * Every command is one entry here, under the identity `commanding.ts` gives
 * it, so a command that is offered and a command that happens are the same
 * list. Nothing here draws anything.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { Deed, Runnable, Shown } from './commanding'
import type { Opened } from './putting'
import type {
  Answer,
  Movement,
  Outcome,
  Refused,
  Removed,
  Renamed,
  VaultRefused,
  Vaults,
} from './core'
import type { Made } from './note/creating'
import type { Says } from './telling'
import { AGENT, FILES, NOTE, PLEX, SETTINGS } from './workspace'

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
  made(path: string, title: string, type: Opened, showing: 'here' | 'beside'): void
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

/** The vault as a command changes what it holds. */
export interface Filing {
  /** A note made under the name it is given, in a seat of another one. */
  makes(title: string, from: string, seat: PlexRelatedSeat | null): Promise<Made | null>
  /** A note given a different name, and its file renamed with it where the two are one name. */
  renames(path: string, title: string): Promise<Renamed>
  /** A note taken out of the vault, into the trash or off the disk. */
  removes(path: string, destroy: boolean): Promise<Removed>
  /**
   * A file or a folder filed somewhere else. The last segment of `to` is what
   * it is called from now on.
   */
  moves(from: string, to: string): Promise<Movement>
  /** An empty folder. The folders above it are made with it. */
  makesFolder(path: string): Promise<Refused | null>
}

/** The runs a command asks for over a file. */
export interface Running {
  /** A recording transcribed, and the words of it written down. */
  transcribes(path: string): Promise<Outcome>
  /** A scanned document read, and the text of it written down. */
  recognises(path: string): Promise<Outcome>
  /** The transcript of a recording put right by a proofreader. */
  proofreads(path: string): Promise<Outcome>
  /**
   * The transcript of a recording taken away, with everything cut from it, and
   * whether this build can do it at all. The recording is left saying nothing,
   * and it is offered to be heard again.
   */
  drops(path: string): Promise<boolean>
}

/** The files a command makes from nothing, each put in front of the person. */
export interface Cutting {
  /**
   * A deck made in a folder under the name it is given. The path it landed at,
   * and nothing where none was made.
   */
  cuts(folder: string, name: string): Promise<string>
  /** A stencil made the same way. */
  stencils(folder: string, name: string): Promise<string>
  /** A preset made the same way, naming none of its settings. */
  presets(folder: string, name: string): Promise<string>
}

/** The vaults this installation holds, as a command changes which one shows. */
export interface Vaulting extends Vaults {
  /** The vault the window is showing, under the name it has now. */
  calls(vault: Shown): void
  /**
   * The page drawn again, on the vault the window shows now. Every tab and
   * every plex belonged to the vault that has gone.
   */
  reloads(): void
}

/** Where a command takes the window. */
export interface Going {
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
export interface Setting {
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
export interface Doing {
  readonly files: Filing
  readonly runs: Running
  readonly cards: Cutting
  readonly vaults: Vaulting
  readonly goes: Going
  readonly settings: Setting
  /** The open files a command reaches, whichever store holds each. */
  readonly notes: Notes
  /** The runs this window has been told this build cannot do. */
  readonly runnable: Runnable
  /** A path put on the clipboard. */
  copies(path: string): void
  /**
   * What was done, or could not be, in words a person reads. One command's
   * word replaces the last, and nothing said clears it.
   */
  readonly says: Says
}

/** Everything carrying a command out says in the window's voice. */
export interface Words {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<Refused, string>
  /** What the list of vaults refused, in words a person reads. */
  readonly unvaulted: Record<VaultRefused, string>
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
}

/** One command, carried out. */
type Carries = (deed: Deed, on: Doing, words: Words) => Promise<void> | void

/** What each command comes to. A command with no entry here does nothing. */
const carried: Record<string, Carries> = {
  read: (deed, on) => on.notes.opens(deed.path, deed.title, 'here'),
  beside: (deed, on) => on.notes.opens(deed.path, deed.title, 'beside'),
  travel: (deed, on) => on.goes.travel(deed.path),
  child: (deed, on, words) => makes(deed, 'child', on, words),
  parent: (deed, on, words) => makes(deed, 'parent', on, words),
  jump: (deed, on, words) => makes(deed, 'jump', on, words),
  note: (deed, on, words) => makes(deed, null, on, words),
  deck: async (deed, on) => {
    if (deed.name) await on.cards.cuts('', deed.name)
  },
  stencil: async (deed, on) => {
    if (deed.name) await on.cards.stencils('', deed.name)
  },
  newPreset: async (deed, on) => {
    if (deed.name) await on.cards.presets('', deed.name)
  },
  title: (deed, on, words) => renames(deed, on, words),
  remove: (deed, on, words) => removes(deed, false, on, words),
  destroy: (deed, on, words) => removes(deed, true, on, words),
  transcribe: async (deed, on, words) =>
    began(deed, await on.runs.transcribes(deed.file), on, words),
  recognise: async (deed, on, words) =>
    began(deed, await on.runs.recognises(deed.file), on, words),
  proofread: async (deed, on, words) =>
    began(deed, await on.runs.proofreads(deed.file), on, words),
  dropTranscript: async (deed, on, words) => {
    if (await on.runs.drops(deed.file)) return
    on.runnable.cannotRun(deed.id)
    on.says(words.unrunnable, 'refusal')
  },
  ask: (deed, on) => on.goes.asks(`${deed.path} — `),
  copy: (deed, on) => on.copies(deed.path),
  reveal: (deed, on) => on.goes.reveals(deed.path),
  preset: (deed, on) => on.goes.preset(deed.path),
  settings: (_, on) => on.goes.opens(SETTINGS),
  move: (deed, on, words) => moves(deed, on, words),
  makeFolder: (deed, on, words) => makesFolder(deed, on, words),
  plex: (_, on) => on.goes.opens(PLEX),
  files: (_, on) => on.goes.opens(FILES),
  agent: (_, on) => on.goes.opens(AGENT),
  close: (deed, on) => on.goes.closes(deed.tab),
  find: (_, on) => on.goes.searches(),
  appearance: (deed, on) => on.settings.appearance(deed.name),
  mode: (deed, on) => on.settings.appearance(deed.name),
  interfaceScale: (deed, on) => on.settings.appearance(deed.name),
  textScale: (deed, on) => on.settings.appearance(deed.name),
  syncing: (deed, on) => on.settings.syncing(deed.name),
  hanging: (deed, on) => on.settings.hanging(deed.name),
  parts: (deed, on) => on.settings.parts(deed.name),
  first: (_, on, words) => travels(on.goes.opening(), on, words),
  goto: (deed, on, words) => travels(deed.path, on, words),
  openVault: (deed, on, words) => shows(deed.vault.id, on, words),
  newVault: (_, on, words) => adds(on, words),
  renameVault: (deed, on, words) => calls(deed, on, words),
  forgetVault: (deed, on, words) => forgets(deed, false, on, words),
  eraseVault: (deed, on, words) => forgets(deed, true, on, words),
}

/** A command carried out. Nothing chosen does nothing at all. */
export async function does(deed: Deed | null, on: Doing, words: Words): Promise<void> {
  if (!deed) return
  const carry = carried[deed.id]
  if (!carry) return
  on.says('')
  try {
    await carry(standing(deed, on), on, words)
  } catch (error) {
    on.says(String(error), 'refusal')
  }
}

/**
 * The deed at the file its note stands at now. One over a note no tab of the
 * window holds is at the name it was made over.
 */
const standing = (deed: Deed, on: Doing): Deed =>
  deed.note ? { ...deed, path: on.notes.where(deed.note) } : deed

/** A tab asked to settle: which one it was, and whether it is still waiting. */
interface Settled {
  readonly held: string | null
  readonly waiting: boolean
}

/**
 * The tab holding a note, once nothing of the note is on its way to its file.
 * A tab waiting on the person to answer for it settles nothing and says so.
 */
const settles = async (path: string, on: Doing): Promise<Settled> => {
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
  deed: Deed,
  seat: PlexRelatedSeat | null,
  on: Doing,
  words: Words,
): Promise<void> => {
  if (!deed.name) return
  const made = await on.files.makes(deed.name, seat ? deed.path : '', seat)
  if (!made) return
  if (deed.kind === NOTE) return on.notes.made(made.path, made.title, 'note', 'beside')
  await travels(made.path, on, words)
}

/**
 * A note given a different name, and its file renamed with it where the two are
 * one name. Prose on disk that nobody here has seen leaves the note as it is.
 */
const renames = async (deed: Deed, on: Doing, words: Words): Promise<void> => {
  if (!deed.name || deed.name === deed.title) return
  const tab = await settles(deed.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.renames(deed.path, deed.name)
  if (answer.changed) return on.says(words.overtaken, 'caution')
  if (answer.refusal) on.says(words.refused[answer.refusal], 'refusal')
}

/**
 * A file or a folder filed somewhere else, carrying the name the path ends in.
 * A destination that is taken leaves it where it was.
 */
const moves = async (deed: Deed, on: Doing, words: Words): Promise<void> => {
  if (!deed.name || deed.name === deed.path) return
  const tab = await settles(deed.path, on)
  if (tab.waiting) return on.says(words.unanswered, 'caution')
  const answer = await on.files.moves(deed.path, deed.name)
  if (answer.refusal === 'occupied') return on.says(words.occupied, 'refusal')
  if (answer.refusal) on.says(words.refused[answer.refusal], 'refusal')
}

/** The outcomes a run is under way in, which the window says as what it did. */
const UNDER_WAY: readonly Answer[] = ['started', 'queued']

/**
 * A run asked for over a file. The application answers how it came out in one
 * sentence, which is what the person is told; a run under way shows what it is
 * doing in the work behind the window. A build that cannot do the run at all is
 * told once and offers it nowhere after that.
 */
const began = (deed: Deed, outcome: Outcome, on: Doing, words: Words): void => {
  if (!outcome.able) {
    on.runnable.cannotRun(deed.id)
    return on.says(words.unrunnable, 'refusal')
  }
  on.says(outcome.why, UNDER_WAY.includes(outcome.answer) ? 'report' : 'refusal')
}

/** An empty folder, made under the path that was typed. */
const makesFolder = async (deed: Deed, on: Doing, words: Words): Promise<void> => {
  if (!deed.name) return
  const refusal = await on.files.makesFolder(deed.name)
  if (refusal === 'occupied') return on.says(words.occupied, 'refusal')
  if (refusal) on.says(words.refused[refusal], 'refusal')
}

/**
 * The files a deed is over: the one it names, and the rest of the selection it
 * was asked over.
 */
const over = (deed: Deed): readonly string[] => [deed.path, ...deed.others]

/**
 * Files taken out of the vault. A file that has gone is gone from the tree, so
 * the notes left pointing at nothing are the whole of what is said. One the
 * vault refuses leaves the rest to go, and what was refused is what the person
 * is told.
 */
const removes = async (deed: Deed, destroy: boolean, on: Doing, words: Words): Promise<void> => {
  const dangling: string[] = []
  const refused: string[] = []
  let waiting = false
  const opening = on.goes.opening()

  for (const path of over(deed)) {
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
const shows = async (id: string, on: Doing, words: Words): Promise<void> => {
  if (!id) return
  const refusal = await on.vaults.open(id)
  if (refusal) return on.says(words.unvaulted[refusal], 'refusal')
  on.vaults.reloads()
}

/**
 * A folder chosen on this machine, added as a vault and opened. A person who
 * chose no folder has asked for nothing.
 */
const adds = async (on: Doing, words: Words): Promise<void> => {
  const path = await on.vaults.choose(words.folder)
  if (!path) return
  const answer = await on.vaults.add(path, '')
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (!answer.vault) return
  await shows(answer.vault.id, on, words)
}

/** A vault called something else. Its folder keeps the name it has on disk. */
const calls = async (deed: Deed, on: Doing, words: Words): Promise<void> => {
  if (!deed.name || deed.name === deed.vault.name) return
  const answer = await on.vaults.rename(deed.vault.id, deed.name)
  if (answer.refusal) return on.says(words.unvaulted[answer.refusal], 'refusal')
  if (answer.vault) on.vaults.calls({ id: answer.vault.id, name: answer.vault.name })
}

/**
 * A vault taken off the list. Erasing it puts the folder in the trash this
 * machine keeps; forgetting it leaves the folder where it is.
 */
const forgets = async (deed: Deed, erase: boolean, on: Doing, words: Words): Promise<void> => {
  const id = deed.vault.id
  const refusal = erase ? await on.vaults.erase(id) : await on.vaults.forget(id)
  if (refusal) on.says(words.unvaulted[refusal], 'refusal')
}

/** A note travelled to, and a vault with none to travel to said. */
const travels = async (path: string, on: Doing, words: Words): Promise<void> => {
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
