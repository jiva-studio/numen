/**
 * Carrying out a command, over what the window offers to do it with.
 *
 * Every command is one entry here, under the identity `commanding.ts` gives
 * it, so a command that is offered and a command that happens are the same
 * list. Nothing here draws anything.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { Deed } from './commanding'
import type { Refused, Removed, Renamed } from './core'
import type { Made } from './note/creating'
import { AGENT, NOTE, PLEX } from './workspace'

/** The notes the window has open, as a command reaches them. */
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
  /** A note put in front of the person, in a tab of its own or one beside it. */
  shows(path: string, title: string, showing: 'here' | 'beside'): void
}

/** What the window offers a command being carried out. */
export interface Doing {
  /** A note made under the name it is given, in a seat of another one. */
  makes(title: string, from: string, seat: PlexRelatedSeat | null): Promise<Made | null>
  /** A note given a different name, and its file renamed with it. */
  renames(path: string, title: string): Promise<Renamed>
  /** A note taken out of the vault, into the trash or off the disk. */
  removes(path: string, destroy: boolean): Promise<Removed>
  readonly notes: Notes
  /** A note put in front of the person, in the plex they are looking at. */
  travel(path: string): Promise<void>
  /** Every plex standing on a note travels to another one. */
  leaves(from: string, to: string): Promise<void>
  /** The note the vault opens with. */
  opening(): string
  /** A tab of a kind, opened and put in front. */
  opens(kind: string): void
  /** A tab let go of. */
  closes(tab: string): void
  /** Something to ask, put in the agent the person was last in. */
  asks(text: string): void
  /** A path put on the clipboard. */
  copies(path: string): void
  /** The search, in place of the commands. */
  searches(): void
  /**
   * The window drawn another way: a theme worn from now on, or which half of a
   * colour pair the tokens are read as. The identity is the window's own.
   */
  appearance(chosen: string): Promise<void>
  /** What was done, or could not be, in words a person reads. */
  says(text: string): void
}

/** Everything carrying a command out says in the window's voice. */
export interface Words {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<Refused, string>
  /** The links that mean another note now, which nothing repairs. */
  readonly retargeted: string
  /** The links in other people's notes that were written again. */
  readonly repaired: string
  /** The links that reach nothing now, which nothing repairs either. */
  readonly dangling: string
  /** Where a removed note landed, which is the only way back to it. */
  readonly trashedAt: string
  /** The rename wrote in the frontmatter, which is the person's own. */
  readonly titled: string
  /** The vault opens with no note at all. */
  readonly nowhere: string
  /** The note is waiting on the person, and its file stays where it is. */
  readonly unanswered: string
  /** The note holds prose nobody here has seen, so nothing was written. */
  readonly overtaken: string
}

/** One command, carried out. */
type Carries = (deed: Deed, on: Doing, words: Words) => Promise<void> | void

/** What each command comes to. A command with no entry here does nothing. */
const carried: Record<string, Carries> = {
  read: (deed, on) => on.notes.shows(deed.path, deed.title, 'here'),
  beside: (deed, on) => on.notes.shows(deed.path, deed.title, 'beside'),
  travel: (deed, on) => on.travel(deed.path),
  child: (deed, on, words) => makes(deed, 'child', on, words),
  parent: (deed, on, words) => makes(deed, 'parent', on, words),
  jump: (deed, on, words) => makes(deed, 'jump', on, words),
  note: (deed, on, words) => makes(deed, null, on, words),
  title: (deed, on, words) => renames(deed, on, words),
  remove: (deed, on, words) => removes(deed, false, on, words),
  destroy: (deed, on, words) => removes(deed, true, on, words),
  ask: (deed, on) => on.asks(`${deed.path} — `),
  copy: (deed, on) => on.copies(deed.path),
  plex: (_, on) => on.opens(PLEX),
  agent: (_, on) => on.opens(AGENT),
  close: (deed, on) => on.closes(deed.tab),
  find: (_, on) => on.searches(),
  appearance: (deed, on) => on.appearance(deed.name),
  first: (_, on, words) => travels(on.opening(), on, words),
  goto: (deed, on, words) => travels(deed.path, on, words),
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
    on.says(String(error))
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
  const made = await on.makes(deed.name, seat ? deed.path : '', seat)
  if (!made) return
  if (deed.kind === NOTE) return on.notes.shows(made.path, made.title, 'beside')
  await travels(made.path, on, words)
}

/** A note given a different name, and whatever that did to the links reported. */
const renames = async (deed: Deed, on: Doing, words: Words): Promise<void> => {
  if (!deed.name || deed.name === deed.title) return
  const tab = await settles(deed.path, on)
  if (tab.waiting) return on.says(words.unanswered)
  const answer = await on.renames(deed.path, deed.name)
  if (answer.changed) return on.says(words.overtaken)
  if (answer.refusal) return on.says(words.refused[answer.refusal])
  const moved = answer.moved
  on.says(
    all(
      answer.frontmatter ? words.titled : '',
      naming(words.repaired, moved?.repaired ?? []),
      naming(words.retargeted, (moved?.retargeted ?? []).map((one) => one.in)),
    ),
  )
}

/** A note taken out of the vault, and where it went and what it left reported. */
const removes = async (deed: Deed, destroy: boolean, on: Doing, words: Words): Promise<void> => {
  const tab = await settles(deed.path, on)
  if (tab.waiting) return on.says(words.unanswered)
  const answer = await on.removes(deed.path, destroy)
  if (answer.refusal) return on.says(words.refused[answer.refusal])
  if (tab.held) on.notes.shuts(tab.held)
  on.says(
    all(
      answer.trashed ? `${words.trashedAt} ${answer.trashed}` : '',
      naming(words.dangling, answer.dangling),
    ),
  )
  const opening = on.opening()
  if (opening) await on.leaves(deed.path, opening)
}

/** A note travelled to, and a vault with none to travel to said. */
const travels = async (path: string, on: Doing, words: Words): Promise<void> => {
  if (!path) return on.says(words.nowhere)
  await on.travel(path)
}

/** What a command did to other notes, named once each under what it did. */
const naming = (says: string, notes: readonly string[]): string => {
  const named = [...new Set(notes)]
  return named.length === 0 ? '' : `${says} ${named.join(', ')}`
}

/** Everything one command has to say, as the one line the window says it in. */
const all = (...says: readonly string[]): string => says.filter((one) => one).join('. ')
