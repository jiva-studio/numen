/**
 * What a window holding nothing offers, asked without a screen.
 *
 * The negatives are the ones worth having: a window over no vault offers no
 * way into one, a vault that never opened offers no new note, and a key is
 * drawn on a row only where the table binds that letter to that command.
 */
import { describe, expect, it } from 'vitest'
import { keyChord } from '@numen/ui'
import type { VaultList, Vault } from '../../shared/core'
import { keyOf } from '../../features/command-palette/chords'
import { COMMANDS, SETTINGS, vaultsOn, waysIn, type ShownVault, type Words } from './screen'
import { WORDS as own } from './words'

const APPLE = 'MacIntel'
const LINUX = 'Linux x86_64'

/** What the screen says, each word standing for itself. */
const words: Words = {
  find: 'find',
  findKeys: keyChord('k', APPLE),
  commands: 'commands',
  commandsKeys: keyChord('p', APPLE),
  newNote: 'new note',
  newAgent: 'new agent',
  vaults: 'vaults',
  current: 'current',
  gone: 'gone',
  settings: 'settings',
  newVault: 'new vault',
  newVaultDetail: 'choose a folder',
}

/** A window showing a vault it has read. */
const at = (over: Partial<ShownVault> = {}): ShownVault => ({
  vault: 'physics',
  ready: true,
  ...over,
})

const vault = (id: string, name: string, missing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  missing,
})

const listed = (vaults: readonly Vault[], showing: string): VaultList => ({ vaults, showing })

/** Two vaults, the first of them the one the window is showing. */
const two = listed([vault('a', 'Physics'), vault('b', 'Heat')], 'a')

describe('the ways into the vault', () => {
  it('are the settings alone where the window is showing no vault', () => {
    expect(waysIn(at({ vault: '', ready: false }), words, APPLE).map((one) => one.id)).toStrictEqual(
      [SETTINGS],
    )
  })

  it('are the settings alone where a window showing no vault says it is ready', () => {
    expect(waysIn(at({ vault: '' }), words, APPLE).map((one) => one.id)).toStrictEqual([SETTINGS])
  })

  it('are the six of a vault that has been read, in the order they are drawn', () => {
    const ways = waysIn(at(), words, APPLE)
    expect(ways.map((one) => one.id)).toStrictEqual([
      'find',
      COMMANDS,
      'note',
      'plex',
      'agent',
      SETTINGS,
    ])
  })

  it('are each called what the window calls them, and the plex what the screen does', () => {
    const ways = waysIn(at(), words, APPLE)
    expect(ways.map((one) => one.text)).toStrictEqual([
      words.find,
      words.commands,
      words.newNote,
      own.plex,
      words.newAgent,
      words.settings,
    ])
  })

  /** A note cannot be made in a vault that never opened. The rest stand. */
  it('leave out the new note where the vault could not be opened', () => {
    const ways = waysIn(at({ ready: false }), words, APPLE)
    expect(ways.map((one) => one.id)).toStrictEqual(['find', COMMANDS, 'plex', 'agent', SETTINGS])
  })

  /** The settings are the installation's, so they are offered last of all. */
  it('offer the settings last, on a vault and off one', () => {
    expect(waysIn(at(), words, APPLE).at(-1)?.id).toBe(SETTINGS)
    expect(waysIn(at({ vault: '' }), words, APPLE).at(-1)?.id).toBe(SETTINGS)
  })
})

describe('the keystroke drawn on a way', () => {
  const keysOn = (id: string, agent = APPLE) =>
    waysIn(at(), words, agent).find((one) => one.id === id)?.keys

  it('is the one the table binds to that command', () => {
    expect(keysOn('note')).toStrictEqual(keyOf('note', APPLE))
    expect(keysOn('plex')).toStrictEqual(keyOf('plex', APPLE))
    expect(keysOn('agent')).toStrictEqual(keyOf('agent', APPLE))
  })

  it('is the key the keyboard in hand holds', () => {
    expect(keysOn('note', LINUX)).toStrictEqual({ icons: ['control'], letter: 'N' })
    expect(keysOn('agent', LINUX)).toStrictEqual({ icons: ['control', 'shift'], letter: 'A' })
  })

  it('is the two the window keeps for itself, as the window hands them over', () => {
    expect(keysOn('find')).toBe(words.findKeys)
    expect(keysOn(COMMANDS)).toBe(words.commandsKeys)
  })

})

describe('the vaults the screen lists', () => {
  it('holds every vault on the list, under its name', () => {
    expect(vaultsOn(two, words).map((one) => one.name)).toStrictEqual(['Physics', 'Heat'])
  })

  it('marks the one the window is showing, and draws it all the same', () => {
    expect(vaultsOn(two, words)[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.current,
    })
  })

  it('marks one whose folder is not there', () => {
    const held = vaultsOn(listed([vault('b', 'Heat', true)], 'a'), words)
    expect(held[0]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
      detail: words.gone,
    })
  })

  it('says the folder is gone of the vault it is showing, which is the worse news', () => {
    const held = vaultsOn(listed([vault('a', 'Physics', true)], 'a'), words)
    expect(held[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.gone,
    })
  })

  it('says nothing beside a vault that is neither', () => {
    expect(vaultsOn(two, words)[1]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
    })
  })

  it('is empty where the installation holds no vault', () => {
    expect(vaultsOn(listed([], ''), words)).toStrictEqual([])
  })
})
