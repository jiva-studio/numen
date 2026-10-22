/**
 * What a window holding nothing offers, asked without a screen.
 *
 * The negatives are the ones worth having: a window over no vault offers no
 * way into one, a vault that never opened offers no new note, and a key is
 * drawn on a row only where the table binds that letter to that command.
 */
import { describe, expect, it } from 'vitest'
import { keyChord } from '@numen/ui'
import type { VaultList, Vault } from '@/entities/vault'
import { findChordKeys } from '@/features/command-palette'
import {
  COMMANDS,
  SETTINGS,
  createVaultRows,
  createWelcomeActions,
  type CurrentVault,
  type Words,
} from './screen'
import { WORDS as own } from '../words'

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
const createCurrentVault = (over: Partial<CurrentVault> = {}): CurrentVault => ({
  vault: 'physics',
  isReady: true,
  ...over,
})

const vault = (id: string, name: string, isMissing = false): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: isMissing,
})

const createVaultList = (vaults: readonly Vault[], current: string): VaultList => ({
  vaults,
  showing: current,
})

/** Two vaults, the first of them the one the window is showing. */
const two = createVaultList([vault('a', 'Physics'), vault('b', 'Heat')], 'a')

describe('the ways into the vault', () => {
  it('are the settings alone where the window is showing no vault', () => {
    expect(
      createWelcomeActions(createCurrentVault({ vault: '', isReady: false }), words, APPLE).map(
        (one) => one.id,
      ),
    ).toStrictEqual([SETTINGS])
  })

  it('are the settings alone where a window showing no vault says it is ready', () => {
    expect(
      createWelcomeActions(createCurrentVault({ vault: '' }), words, APPLE).map((one) => one.id),
    ).toStrictEqual([SETTINGS])
  })

  it('are the six of a vault that has been read, in the order they are drawn', () => {
    const ways = createWelcomeActions(createCurrentVault(), words, APPLE)
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
    const ways = createWelcomeActions(createCurrentVault(), words, APPLE)
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
    const ways = createWelcomeActions(createCurrentVault({ isReady: false }), words, APPLE)
    expect(ways.map((one) => one.id)).toStrictEqual(['find', COMMANDS, 'plex', 'agent', SETTINGS])
  })

  /** The settings are the installation's, so they are offered last of all. */
  it('offer the settings last, on a vault and off one', () => {
    expect(createWelcomeActions(createCurrentVault(), words, APPLE).at(-1)?.id).toBe(SETTINGS)
    expect(createWelcomeActions(createCurrentVault({ vault: '' }), words, APPLE).at(-1)?.id).toBe(
      SETTINGS,
    )
  })
})

describe('the keystroke drawn on a way', () => {
  const getKeys = (id: string, agent = APPLE) =>
    createWelcomeActions(createCurrentVault(), words, agent).find((one) => one.id === id)?.keys

  it('is the one the table binds to that command', () => {
    expect(getKeys('note')).toStrictEqual(findChordKeys('note', APPLE))
    expect(getKeys('plex')).toStrictEqual(findChordKeys('plex', APPLE))
    expect(getKeys('agent')).toStrictEqual(findChordKeys('agent', APPLE))
  })

  it('is the key the keyboard in hand holds', () => {
    expect(getKeys('note', LINUX)).toStrictEqual({ icons: ['control'], letter: 'N' })
    expect(getKeys('agent', LINUX)).toStrictEqual({ icons: ['control', 'shift'], letter: 'A' })
  })

  it('is the two the window keeps for itself, as the window passes them over', () => {
    expect(getKeys('find')).toBe(words.findKeys)
    expect(getKeys(COMMANDS)).toBe(words.commandsKeys)
  })
})

describe('the vaults the screen lists', () => {
  it('holds every vault on the list, under its name', () => {
    expect(createVaultRows(two, words).map((one) => one.name)).toStrictEqual(['Physics', 'Heat'])
  })

  it('marks the one the window is showing, and draws it all the same', () => {
    expect(createVaultRows(two, words)[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.current,
    })
  })

  it('marks one whose folder is not there', () => {
    const target = createVaultRows(createVaultList([vault('b', 'Heat', true)], 'a'), words)
    expect(target[0]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
      detail: words.gone,
    })
  })

  it('says the folder is gone of the vault it is showing, which is the worse news', () => {
    const target = createVaultRows(createVaultList([vault('a', 'Physics', true)], 'a'), words)
    expect(target[0]).toStrictEqual({
      id: 'a',
      name: 'Physics',
      path: '/vaults/Physics',
      detail: words.gone,
    })
  })

  it('says nothing alongside a vault that is neither', () => {
    expect(createVaultRows(two, words)[1]).toStrictEqual({
      id: 'b',
      name: 'Heat',
      path: '/vaults/Heat',
    })
  })

  it('is empty where the installation holds no vault', () => {
    expect(createVaultRows(createVaultList([], ''), words)).toStrictEqual([])
  })
})
