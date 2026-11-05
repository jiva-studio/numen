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
    commands: readonly { id: string; title: string; scope?: string }[]
    menus?: readonly { location: string; command: string; group?: string; order?: number }[]
  }
}

describe('the manifest', () => {
  it('declares vault:write, which the notes the graph joins and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places showing a note in the plex in the menu of a search result, among the navigation entries', () => {
    expect(manifest.contributes.menus).toStrictEqual([
      { location: 'searchResult', command: 'graph.showNote', group: 'navigate', order: 20 },
    ])
  })

  it('keeps the declaration of the command that shows a note, with no field about its place', () => {
    const command = manifest.contributes.commands.find((one) => one.id === 'graph.showNote')

    expect(command).toMatchObject({ title: 'Show note in plex', scope: 'note' })
    expect(Object.keys(command ?? {})).not.toContain('location')
    expect(Object.keys(command ?? {})).not.toContain('menus')
  })
})
