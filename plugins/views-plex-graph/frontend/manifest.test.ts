/**
 * The permissions the plugin's manifest declares, which the window holds every call to, and the
 * menus it places its commands in.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  permissions: readonly string[]
  contributes: {
    commands: readonly { id: string; title: string; icon?: string; scope?: string }[]
    menus?: readonly { location: string; command: string; group?: string; order?: number }[]
    slots?: readonly Record<string, unknown>[]
  }
}

/** The entries the plugin places in the menu of a row of the file tree and in its submenu there. */
const getMenuEntries = () =>
  (manifest.contributes.slots ?? []).filter(
    (one) => one.slot === 'fileTree.context' || one.slot === 'plex.relation',
  )

describe('the manifest', () => {
  it('declares vault:write, which the notes the graph joins and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places showing a note in the menu of a search result and opening the plex in the title bar, both among the navigation entries', () => {
    expect(manifest.contributes.menus).toStrictEqual([
      { location: 'searchResult', command: 'graph.showNote', group: 'navigate', order: 20 },
      { location: 'titleBar.commands', command: 'graph.open', group: 'navigate', order: 20 },
    ])
  })

  it('declares the command the title bar shows with the title and the icon of a graph', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'graph.open')).toMatchObject({
      title: 'Open plex graph',
      icon: 'waypoints',
    })
  })

  it('keeps the command that shows a note out of the title bar, which has no note', () => {
    const entries = (manifest.contributes.menus ?? []).filter(
      (one) => one.location === 'titleBar.commands',
    )

    expect(entries.map((one) => one.command)).not.toContain('graph.showNote')
  })

  it('keeps the declaration of the command that shows a note, with no field about its place', () => {
    const command = manifest.contributes.commands.find((one) => one.id === 'graph.showNote')

    expect(command).toMatchObject({ title: 'Show note in plex', scope: 'note' })
    expect(Object.keys(command ?? {})).not.toContain('location')
    expect(Object.keys(command ?? {})).not.toContain('menus')
  })
})

describe('the menu of a row of the file tree', () => {
  it('places showing the note, and the submenu of relations with its three commands', () => {
    expect(getMenuEntries()).toStrictEqual([
      { slot: 'fileTree.context', command: 'graph.showNote', group: '0-open', order: 20 },
      {
        slot: 'fileTree.context',
        submenu: 'plex.relation',
        title: 'Relation',
        group: '4-relation',
      },
      { slot: 'plex.relation', command: 'relation.createChild', order: 10 },
      { slot: 'plex.relation', command: 'relation.createParent', order: 20 },
      { slot: 'plex.relation', command: 'relation.createJump', order: 30 },
    ])
  })

  it('declares the commands of the relations over a note', () => {
    const byId = (id: string) => manifest.contributes.commands.find((one) => one.id === id)

    expect(byId('relation.createChild')).toStrictEqual({
      id: 'relation.createChild',
      title: 'New child note',
      scope: 'note',
    })
    expect(byId('relation.createParent')).toStrictEqual({
      id: 'relation.createParent',
      title: 'New parent note',
      scope: 'note',
    })
    expect(byId('relation.createJump')).toStrictEqual({
      id: 'relation.createJump',
      title: 'New jump note',
      scope: 'note',
    })
  })
})

describe('the readme', () => {
  it('names the title bar button that opens the plex', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
