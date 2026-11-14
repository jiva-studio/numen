/**
 * The permissions the plugin's manifest declares, which the window holds every call to, and the
 * slots it places its commands in.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { JsonValue } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { readLook } from './model/look'
import { readNames } from './model/names'
import { readNesting } from './model/nesting'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  permissions: readonly string[]
  contributes: {
    commands: readonly {
      id: string
      title: string
      icon?: string
      scope?: string
      when?: string
    }[]
    keybindings?: readonly { key: string; command: string }[]
    slots?: readonly Record<string, unknown>[]
  }
}

const getMenuEntries = () =>
  (manifest.contributes.slots ?? []).filter(
    (one) => one.slot === 'fileTree.context' || one.slot === 'plex.relation',
  )

describe('the manifest', () => {
  it('declares vault:write, which the notes the graph joins and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places showing a note in the menu of a search result and opening the plex in the title bar, both among the navigation entries', () => {
    expect(
      (manifest.contributes.slots ?? []).filter(
        (one) => one.slot === 'palette.searchResult' || one.slot === 'titleBar.commands',
      ),
    ).toStrictEqual([
      {
        slot: 'palette.searchResult',
        command: 'app:file.reveal',
        values: { kind: 'plex' },
        title: 'Show in plex',
        icon: 'waypoints',
        group: 'navigate',
        order: 20,
        when: 'target.isNote',
      },
      { slot: 'titleBar.commands', command: 'graph.open', group: 'navigate', order: 20 },
    ])
  })

  it('places showing the note in front in the plex among the commands of the palette over a note', () => {
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'palette.note'),
    ).toStrictEqual([
      {
        slot: 'palette.note',
        command: 'app:file.reveal',
        values: { kind: 'plex' },
        title: 'Show in plex',
        icon: 'waypoints',
        when: 'target.isNote',
      },
    ])
  })

  it('declares the command the title bar shows with the title and the icon of a graph', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'graph.open')).toMatchObject({
      title: 'Open plex graph',
      icon: 'waypoints',
    })
  })

  it('keeps the command that shows a note out of the title bar, which has no note', () => {
    const entries = (manifest.contributes.slots ?? []).filter(
      (one) => one.slot === 'titleBar.commands',
    )

    expect(entries.map((one) => one.command)).not.toContain('app:file.reveal')
  })

  it('declares no command to show a note, which the window runs as app:file.reveal over the plex', () => {
    expect(manifest.contributes.commands.map((one) => one.id)).toStrictEqual(['graph.open'])
  })
})

describe('the shortcut', () => {
  it('opens the plex on the window key with Shift and X', () => {
    expect(manifest.contributes.keybindings).toContainEqual({
      key: 'mod+shift+x',
      command: 'graph.open',
    })
  })
})

describe('the menu of a row of the file tree', () => {
  it('places showing the note, and the submenu of relations with the window’s commands', () => {
    expect(getMenuEntries()).toStrictEqual([
      {
        slot: 'fileTree.context',
        command: 'app:file.reveal',
        values: { kind: 'plex' },
        title: 'Show in plex',
        icon: 'waypoints',
        group: '0-open',
        order: 20,
        when: 'target.isNote',
      },
      {
        slot: 'fileTree.context',
        submenu: 'plex.relation',
        title: 'Relation',
        icon: 'waypoints',
        group: '4-relation',
      },
      { slot: 'plex.relation', command: 'app:note.createChild', order: 10 },
      { slot: 'plex.relation', command: 'app:note.createParent', order: 20 },
      { slot: 'plex.relation', command: 'app:note.createJump', order: 30 },
    ])
  })

  it('declares no command of its own for the relations', () => {
    expect(
      manifest.contributes.commands.map((one) => one.id).filter((id) => id.startsWith('relation.')),
    ).toStrictEqual([])
  })
})

describe('the menu on a node', () => {
  it('is the slot plexGraph.context, filled with the commands of the window over a note, and no new note on a node', () => {
    expect(
      (manifest.contributes.slots ?? [])
        .filter((one) => one.slot === 'plexGraph.context' && one.command !== 'app:note.create')
        .map((one) => [one.command, one.group]),
    ).toStrictEqual([
      ['app:file.open', '0-open'],
      ['app:file.reveal', '1-note'],
      ['app:file.copyPath', '2-file'],
      ['app:note.createChild', '3-plex'],
      ['app:note.createParent', '3-plex'],
      ['app:note.createJump', '3-plex'],
      ['app:note.rename', '3-plex'],
      ['app:file.remove', '9-remove'],
    ])
  })
})

describe('the menu of a place with no node', () => {
  const getCanvasEntries = () =>
    (manifest.contributes.slots ?? []).filter(
      (one) => one.slot === 'plexGraph.context' && one.command === 'app:note.create',
    )

  it('declares no command of its own to make a note', () => {
    expect(manifest.contributes.commands.map((one) => one.id)).not.toContain('graph.createNote')
  })

  it('offers the command of the window to make a note over a folder alone, in the group of new rows', () => {
    expect(getCanvasEntries()).toStrictEqual([
      {
        slot: 'plexGraph.context',
        command: 'app:note.create',
        group: '0-new',
        order: 10,
        when: 'target.isFolder',
      },
    ])
  })
})

describe('the look settings of the plex', () => {
  const { settings } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
  ) as { settings: readonly Record<string, unknown>[] }

  it('offers Pill background in the command palette, on by default', () => {
    expect(settings.find((one) => one.key === 'showPills')).toMatchObject({
      type: 'boolean',
      default: true,
      palette: true,
      icon: 'waypoints',
    })
  })
})

describe('the settings the code reads', () => {
  interface Setting {
    key: string
    type: string
    default: unknown
    min?: number
    max?: number
    options?: readonly { value: string }[]
  }
  const { settings } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
  ) as { settings: readonly Setting[] }

  const createContext = (initialSettings: Record<string, JsonValue> = {}) =>
    createMockPluginContext({ pluginId: 'md.numen.views.plex.graph', initialSettings })

  /** The value the code holds for every manifest key, read from a context with these settings. */
  const readValues = (initialSettings: Record<string, JsonValue>): Record<string, unknown> => {
    const context = createContext(initialSettings)
    const look = readLook(context)
    const names = readNames(context)
    const nesting = readNesting(context)
    return {
      ...Object.fromEntries(Object.entries(look).map(([key, ref]) => [key, ref.value])),
      showNames: names.showNames.value,
      namesPerNote: names.namesPerNote.value,
      nestHeadings: nesting.isNesting.value,
      headingLimit: nesting.headings.value,
    }
  }

  it('are all read by the code, and the code reads no other', () => {
    expect(Object.keys(readValues({})).sort()).toStrictEqual(settings.map((one) => one.key).sort())
  })

  it('start at the default the manifest declares', () => {
    const values = readValues({})

    expect(settings.map((one) => [one.key, values[one.key]])).toStrictEqual(
      settings.map((one) => [one.key, one.default]),
    )
  })

  it('take the minimum and the maximum the manifest declares and no value beyond them', () => {
    const ranges = settings.filter((one) => one.type === 'range')
    const getBeyond = (offset: (one: Setting) => number) =>
      readValues(Object.fromEntries(ranges.map((one) => [one.key, offset(one)])))
    const below = getBeyond((one) => (one.min ?? 0) - 1000)
    const above = getBeyond((one) => (one.max ?? 0) + 1000)

    expect(ranges.map((one) => [one.key, below[one.key], above[one.key]])).toStrictEqual(
      ranges.map((one) => [one.key, one.min, one.max]),
    )
  })

  it('take every option of an enum the manifest declares and fall back to the default for another value', () => {
    const enums = settings.filter((one) => one.type === 'enum')

    for (const one of enums) {
      const values = (one.options ?? []).map((option) => option.value)
      const readBack = values.map((value) => readValues({ [one.key]: value })[one.key])

      expect(readBack).toStrictEqual(values)
      expect(readValues({ [one.key]: 'no such option' })[one.key]).toBe(one.default)
    }
  })
})
