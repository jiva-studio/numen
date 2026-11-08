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
      shortcut?: string
    }[]
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
      { slot: 'palette.searchResult', command: 'graph.showNote', group: 'navigate', order: 20 },
      { slot: 'titleBar.commands', command: 'graph.open', group: 'navigate', order: 20 },
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

    expect(entries.map((one) => one.command)).not.toContain('graph.showNote')
  })

  it('keeps the declaration of the command that shows a note, with no field about its place', () => {
    const command = manifest.contributes.commands.find((one) => one.id === 'graph.showNote')

    expect(command).toMatchObject({ title: 'Show note in plex', when: 'target.isNote' })
    expect(Object.keys(command ?? {})).not.toContain('location')
    expect(Object.keys(command ?? {})).not.toContain('slots')
  })
})

describe('the shortcut', () => {
  it('opens the plex on the window key with Shift and X', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'graph.open')?.shortcut).toBe(
      'mod+shift+x',
    )
  })
})

describe('the menu of a row of the file tree', () => {
  it('places showing the note, and the submenu of relations with the window’s commands', () => {
    expect(getMenuEntries()).toStrictEqual([
      { slot: 'fileTree.context', command: 'graph.showNote', group: '0-open', order: 20 },
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
  it('is the slot plexGraph.context, filled with the commands of the window over a note', () => {
    expect(
      (manifest.contributes.slots ?? [])
        .filter((one) => one.slot === 'plexGraph.context')
        .map((one) => [one.command, one.group]),
    ).toStrictEqual([
      ['app:note.open', '0-open'],
      ['graph.showNote', '1-note'],
      ['app:note.copyPath', '2-file'],
      ['app:note.createChild', '3-plex'],
      ['app:note.createParent', '3-plex'],
      ['app:note.createJump', '3-plex'],
      ['app:note.rename', '3-plex'],
      ['app:note.trash', '9-remove'],
    ])
  })
})

describe('the readme', () => {
  it('names the title bar button that opens the plex', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
