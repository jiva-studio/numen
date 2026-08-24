/**
 * Carrying out a command, over what the window offers to do it with.
 *
 * Every command is one entry here, under the identity `commanding.ts` gives
 * it, so a command that is offered and a command that happens are the same
 * list. Nothing here draws anything.
 */
import type { PlexRelatedSeat } from '@numen/ui'
import type { Deed } from './commanding'
import type { Refused, Removed, Renamed, Retargeted } from './core'
import type { Made } from './note/creating'
import { AGENT, NOTE, PLEX } from './workspace'

/** The notes the window has open, as a command reaches them. */
export interface Notes {
  /** The identity of the tab standing at a file, and nothing where none does. */
  holding(path: string): string | null
  /** Answers once nothing of that note is on its way to the file. */
  settles(id: string): Promise<void>
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
  /** The note that plex is standing on. */
  standing(): string
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
  /** What was done, or could not be, in words a person reads. */
  says(text: string): void
}

/** Everything carrying a command out says in the window's voice. */
export interface Words {
  /** What the vault refused, in words a person reads. */
  readonly refused: Record<Refused, string>
  /** The links that mean another note now, which nothing repairs. */
  readonly retargeted: string
  /** The links that reach nothing now, which nothing repairs either. */
  readonly dangling: string
  /** The vault opens with no note at all. */
  readonly nowhere: string
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
    await carry(deed, on, words)
  } catch (error) {
    on.says(String(error))
  }
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
  await settled(deed.path, on)
  const answer = await on.renames(deed.path, deed.name)
  // The note is brought into line before its file is, so a refusal to move the
  // file leaves the note under its new name.
  if (answer.refusal) return on.says(words.refused[answer.refusal])
  on.says(retargeted(answer.moved?.retargeted ?? [], words))
}

/** A note taken out of the vault, and whatever now links to nothing reported. */
const removes = async (
  deed: Deed,
  destroy: boolean,
  on: Doing,
  words: Words,
): Promise<void> => {
  await settled(deed.path, on)
  const answer = await on.removes(deed.path, destroy)
  if (answer.refusal) return on.says(words.refused[answer.refusal])
  on.says(dangling(answer.dangling, words))
  // A plex standing on the note that went is left standing on nothing, so it
  // travels to the note the vault opens with.
  if (on.standing() === deed.path && on.opening()) await on.travel(on.opening())
}

/**
 * Nothing of the note is on its way to its file. The window writes a note over
 * whatever the file holds, and a write that lands after the file has moved
 * writes the note back where it was.
 */
const settled = async (path: string, on: Doing): Promise<void> => {
  const id = on.notes.holding(path)
  if (id) await on.notes.settles(id)
}

/** A note travelled to, and a vault with none to travel to said. */
const travels = async (path: string, on: Doing, words: Words): Promise<void> => {
  if (!path) return on.says(words.nowhere)
  await on.travel(path)
}

/** The links that mean another note now, which are reported and not repaired. */
const retargeted = (links: readonly Retargeted[], words: Words): string => {
  const notes = [...new Set(links.map((one) => one.in))]
  return notes.length === 0 ? '' : `${words.retargeted} ${notes.join(', ')}`
}

/** The links that reach nothing now, which are reported and not repaired. */
const dangling = (notes: readonly string[], words: Words): string =>
  notes.length === 0 ? '' : `${words.dangling} ${notes.join(', ')}`
