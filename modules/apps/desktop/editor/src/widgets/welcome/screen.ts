/**
 * What the window offers while it holds no tab.
 *
 * The screen draws these rows and decides nothing, so a test can ask what a
 * window standing on a vault, or on none, puts in front of the person.
 */
import type { PaletteKeys, VaultRow, WelcomeAction } from '@numen/ui'
import type { VaultList } from '../../shared/core'
import { keysOf } from '../../features/command-palette/chords'
import { WORDS as own } from './words'

/** What the welcome screen is drawn over. */
export interface ShownVault {
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
  /** Everything this installation is configured as. */
  readonly settings: string
  /** The vaults, and what a row of that list carries beside its name. */
  readonly vaults: string
  readonly current: string
  readonly gone: string
  readonly newVault: string
  readonly newVaultDetail: string
}

/** The row that puts the commands up, which is no command of its own. */
export const COMMANDS = 'commands'

/** The row that opens the settings, which stand outside any vault. */
export const SETTINGS = 'settings'

/**
 * The ways in, in the order they are drawn. The settings are the
 * installation's, so they are offered last and are offered always. The rest
 * stand on a vault: a window showing none offers none of them, and the list
 * below is where a person goes first. A note is offered once the vault has
 * been read.
 *
 * Every keystroke drawn here is the one the table binds, so a key a person sees
 * is a key that works.
 */
export const waysIn = (at: ShownVault, words: Words, agent: string): readonly WelcomeAction[] => {
  const settings: WelcomeAction = { id: SETTINGS, text: words.settings, ...keysOf(SETTINGS, agent) }
  if (at.vault === '') return [settings]
  const note: readonly WelcomeAction[] = at.ready
    ? [{ id: 'note', text: words.newNote, ...keysOf('note', agent) }]
    : []
  return [
    { id: 'find', text: words.find, keys: words.findKeys },
    { id: COMMANDS, text: words.commands, keys: words.commandsKeys },
    ...note,
    { id: 'plex', text: own.plex, ...keysOf('plex', agent) },
    { id: 'agent', text: words.newAgent, ...keysOf('agent', agent) },
    settings,
  ]
}

/**
 * The vaults the installation holds. The one in front of the person says so,
 * and so does one whose folder is not there; the rest carry nothing.
 */
export const vaultsOn = (listed: VaultList, words: Words): readonly VaultRow[] =>
  listed.vaults.map((one) => {
    const aside = one.missing ? words.gone : one.id === listed.showing ? words.current : ''
    return {
      id: one.id,
      name: one.name,
      path: one.path,
      ...(aside ? { detail: aside } : {}),
    }
  })
