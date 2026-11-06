/** The places plugins put their commands in, over the commands the window lists. */
import { describe, expect, it, vi } from 'vitest'
import type { MenuPlacement, PluginPaletteCommand } from '@numen/plugin-host'
import { createCommands, type Command } from '@/features/command-palette'
import { COMMAND } from '@/shared/commandIds'
import { WORDS } from '@/shared/words'
import type { DeclaredCommand } from '@/entities/plugin'
import { createPluginMenus, type PluginMenusDeps } from './pluginMenus'

const ATLAS = 'org.example.atlas'

const declared = (id: string, overrides: Partial<DeclaredCommand> = {}): DeclaredCommand => ({
  pluginId: ATLAS,
  id,
  title: id,
  contentTypes: [],
  extensions: [],
  arguments: [],
  ...overrides,
})

const editorCommand = (id: string, text: string, iconName?: string): Command => ({
  id,
  text,
  group: 'window',
  ...(iconName ? { iconName } : {}),
  isAvailable: () => true,
})

function setup(
  placements: readonly MenuPlacement[] | undefined,
  lists: Partial<{
    plugin: readonly PluginPaletteCommand[]
    declared: readonly DeclaredCommand[] | undefined
    editor: readonly Command[]
  }> = {},
) {
  const reportIssue = vi.fn<(pluginId: string, message: string) => void>()
  const deps: PluginMenusDeps = {
    getPlacements: () => placements,
    getPluginCommands: () => lists.plugin ?? [],
    getDeclaredCommands: () => lists.declared ?? [],
    getEditorCommands: () => lists.editor ?? [],
    reportIssue,
  }
  return { menus: createPluginMenus(deps), reportIssue }
}

describe('a command of a plugin frontend in a place', () => {
  it('carries its title, icon and shortcut and files a short id under its plugin', () => {
    const { menus } = setup([{ pluginId: ATLAS, location: 'titleBar.commands', command: 'atlas.open' }], {
      plugin: [
        { id: `${ATLAS}:atlas.open`, title: 'Open atlas', icon: 'map', shortcut: 'mod+shift+m' },
      ],
    })

    expect(menus.getEntries('titleBar.commands')).toEqual([
      {
        pluginId: ATLAS,
        command: `${ATLAS}:atlas.open`,
        group: '',
        title: 'Open atlas',
        icon: 'map',
        shortcut: 'mod+shift+m',
      },
    ])
  })

  it('keeps the scope of a command that needs a note out of the title bar', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar.commands', command: 'atlas.show' },
        { pluginId: ATLAS, location: 'searchResult', command: 'atlas.show' },
      ],
      { plugin: [{ id: `${ATLAS}:atlas.show`, title: 'Show note', icon: 'map', scope: 'note' }] },
    )

    expect(menus.getEntries('titleBar.commands')).toEqual([])
    expect(menus.getEntries('searchResult').map((one) => one.command)).toEqual([
      `${ATLAS}:atlas.show`,
    ])
  })
})

describe('a command that only the backend of a plugin runs, in a place', () => {
  it('is found among the declared commands with its title and icon', () => {
    const { menus } = setup([{ pluginId: ATLAS, location: 'titleBar.commands', command: 'atlas.sync' }], {
      declared: [declared('atlas.sync', { title: 'Sync atlas', icon: 'refresh-cw' })],
    })

    expect(menus.getEntries('titleBar.commands')).toMatchObject([
      { command: `${ATLAS}:atlas.sync`, title: 'Sync atlas', icon: 'refresh-cw' },
    ])
  })

  it('is left out and reported where no list has the command', () => {
    const { menus, reportIssue } = setup(
      [{ pluginId: ATLAS, location: 'titleBar.commands', command: 'atlas.ghost' }],
      { editor: [editorCommand(COMMAND.openSettings, 'Open settings', 'settings')] },
    )

    expect(menus.getEntries('titleBar.commands').map((one) => one.command)).toEqual([COMMAND.openSettings])
    expect(reportIssue).toHaveBeenCalledTimes(1)
  })
})

describe('a command of the editor in a place', () => {
  it('has the title of the command and the icon name it declares, and no shortcut', () => {
    const { menus } = setup(
      [{ pluginId: ATLAS, location: 'titleBar.commands', command: COMMAND.createNote }],
      { editor: [editorCommand(COMMAND.createNote, 'New note', 'file-plus')] },
    )

    const [entry] = menus.getEntries('titleBar.commands')

    expect(entry).toMatchObject({
      command: COMMAND.createNote,
      title: 'New note',
      icon: 'file-plus',
    })
    expect(entry?.shortcut).toBeUndefined()
  })

  it('is left out of the title bar and kept in a search result where it has no icon name', () => {
    const { menus } = setup(
      [
        { pluginId: ATLAS, location: 'titleBar.commands', command: COMMAND.createNote },
        { pluginId: ATLAS, location: 'searchResult', command: COMMAND.createNote },
      ],
      { editor: [editorCommand(COMMAND.createNote, 'New note')] },
    )

    expect(menus.getEntries('titleBar.commands')).toEqual([])
    expect(menus.getEntries('searchResult').map((one) => one.title)).toEqual(['New note'])
  })

  it('is the command of the editor table that creates a note, with the icon file-plus', () => {
    const table = createCommands(WORDS)
    const { menus } = setup(
      [{ pluginId: ATLAS, location: 'titleBar.commands', command: COMMAND.createNote }],
      { editor: table },
    )

    expect(menus.getEntries('titleBar.commands')).toMatchObject([
      { command: COMMAND.openSettings, title: WORDS.openSettings, icon: 'settings' },
      { command: COMMAND.createNote, title: WORDS.newNote, icon: 'file-plus' },
    ])
  })
})

describe('the places before the first list has loaded', () => {
  it('have no entries', () => {
    const { menus, reportIssue } = setup(undefined, { declared: undefined })

    expect(menus.getEntries('titleBar.commands')).toEqual([])
    expect(menus.getEntries('searchResult')).toEqual([])
    expect(reportIssue).not.toHaveBeenCalled()
  })
})
