/**
 * The permissions the plugin's manifest declares, which the window holds every call to, and the
 * slots it places its commands in.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

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

describe('the readme', () => {
  it('names the title bar button that opens the plex', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})

describe('the look settings of the plex', () => {
  const { settings } = JSON.parse(
    readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
  ) as { settings: readonly Record<string, unknown>[] }

  const NUMBERS = [
    ['pillPaddingSides', 'Pill padding, sides', 10, 0, 24, 1],
    ['pillPaddingEnds', 'Pill padding, top and bottom', 9.25, 0, 16, 0.25],
    ['focusSize', 'Focus size', 1.22, 1, 2.5, 0.01],
    ['animationLength', 'Animation', 420, 0, 2000, 20],
    ['lineThickness', 'Line thickness', 2, 0.5, 4, 0.25],
    ['outlineThickness', 'Outline thickness', 1, 0.5, 4, 0.25],
    ['lineCurvature', 'Line curvature', 0.55, 0, 1, 0.05],
    ['noteRoom', 'Room around a note', 0, 0, 40, 1],
  ] as const

  it('offers Pill background in the command palette, on by default', () => {
    expect(settings.find((one) => one.key === 'showPills')).toMatchObject({
      type: 'boolean',
      title: 'Pill background',
      default: true,
      palette: true,
      icon: 'waypoints',
    })
  })

  it('declares every number with its title, default, range and step', () => {
    for (const [key, title, value, min, max, step] of NUMBERS) {
      expect(
        settings.find((one) => one.key === key),
        key,
      ).toMatchObject({
        type: 'range',
        title,
        default: value,
        min,
        max,
        step,
      })
    }
  })

  it('declares the look after the heading and names settings, which keep their keys', () => {
    expect(settings.map((one) => one.key)).toStrictEqual([
      'references',
      'nestHeadings',
      'headingLimit',
      'showNames',
      'namesPerNote',
      'namesOpacity',
      'focusedNamesOpacity',
      'showPills',
      'pillPaddingSides',
      'pillPaddingEnds',
      'focusSize',
      'lineThickness',
      'outlineThickness',
      'lineCurvature',
      'animationLength',
      'noteRoom',
    ])
  })

  it('declares the opacity of the names as two sliders from nothing to one', () => {
    for (const [key, value] of [
      ['namesOpacity', 0.7],
      ['focusedNamesOpacity', 1],
    ] as const) {
      expect(
        settings.find((one) => one.key === key),
        key,
      ).toMatchObject({
        type: 'range',
        default: value,
        min: 0,
        max: 1,
        step: 0.05,
      })
    }
  })

  it('declares References as four choices that show the notes linked in the text, both ways by default', () => {
    const setting = settings.find((one) => one.key === 'references')

    expect(setting).toMatchObject({ type: 'enum', title: 'References', default: 'both' })
    expect((setting?.options as { value: string }[]).map((option) => option.value)).toStrictEqual([
      'off',
      'outgoing',
      'both',
      'incoming',
    ])
  })

  it('gives every look setting a description', () => {
    for (const one of settings) expect(one.description, String(one.key)).toBeTruthy()
  })
})
