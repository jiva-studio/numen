/**
 * What the window offers while it holds no tab.
 *
 * The screen draws these rows and decides nothing, so a test can ask what a
 * window standing on a vault, or on none, puts in front of the person.
 */
import type { PaletteKeys } from '@numen/ui'
import type { Listed } from '../core'
import { keysOf } from '../keying'
import { WORDS as own } from './words'

/** What the welcome screen is drawn over. */
export interface Standing {
  /** The vault the window is showing, and nothing where it shows none. */
  readonly vault: string
  /** Whether that vault has been read and can be asked to do anything. */
  readonly ready: boolean
}

/** Everything the welcome screen says in the window's voice. */
export interface Words {
  /** The palette and the commands, each with the keystroke that puts it up. */
  readonly find: string
  readonly findKeys: PaletteKeys
  readonly commands: string
  readonly commandsKeys: PaletteKeys
  /** What each of the ways into the vault the window names is called. */
  readonly newNote: string
  readonly newAgent: string
  /** The vaults, and what a row of that list carries beside its name. */
  readonly vaults: string
  readonly current: string
  readonly gone: string
  readonly newVault: string
}

/** The row that puts the commands up, which is no command of its own. */
export const COMMANDS = 'commands'

/** One way into the vault: what it asks for, what it is called, its keystroke. */
export interface Way {
  readonly id: string
  readonly text: string
  readonly keys?: PaletteKeys
}

/** One vault of the list, as the screen draws it. */
export interface Held {
  readonly id: string
  readonly name: string
  /** A second line: what is true of this row and not of the ones beside it. */
  readonly detail?: string
}

/**
 * The ways into the vault the window is showing, in the order they are drawn.
 * A window showing none offers no way at all: a plex and an agent each stand
 * on a vault, and the list below is where a person goes first. A note is
 * offered once the vault has been read.
 *
 * Every keystroke drawn here is the one the table binds, so a key a person sees
 * is a key that works.
 */
export const waysIn = (at: Standing, words: Words, agent: string): readonly Way[] => {
  if (at.vault === '') return []
  const note: readonly Way[] = at.ready
    ? [{ id: 'note', text: words.newNote, ...keysOf('note', agent) }]
    : []
  return [
    { id: 'find', text: words.find, keys: words.findKeys },
    { id: COMMANDS, text: words.commands, keys: words.commandsKeys },
    ...note,
    { id: 'plex', text: own.plex, ...keysOf('plex', agent) },
    { id: 'agent', text: words.newAgent, ...keysOf('agent', agent) },
  ]
}

/**
 * The vaults the installation holds. The one in front of the person says so,
 * and so does one whose folder is not there; the rest carry nothing.
 */
export const vaultsOn = (listed: Listed, words: Words): readonly Held[] =>
  listed.vaults.map((one) => {
    const aside = one.missing ? words.gone : one.id === listed.showing ? words.current : ''
    return { id: one.id, name: one.name, ...(aside ? { detail: aside } : {}) }
  })
