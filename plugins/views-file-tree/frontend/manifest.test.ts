/**
 * The permissions the plugin's manifest declares, which the window holds every call to, and the
 * place it takes in the title bar.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  permissions: readonly string[]
  contributes: {
    commands: readonly { id: string; title: string; icon: string }[]
    menus?: readonly { location: string; command: string; group?: string; order?: number }[]
    slots?: readonly Record<string, unknown>[]
  }
}

/** The entries the plugin places in the menu of a row and in the submenu it declares there. */
const getMenuEntries = () =>
  (manifest.contributes.slots ?? []).filter((one) => String(one.slot).startsWith('fileTree.'))

const getCommand = (id: string) => manifest.contributes.commands.find((one) => one.id === id)

describe('the manifest', () => {
  it('declares vault:write, which the files the tree moves and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places opening the file tree first among the navigation entries of the title bar', () => {
    expect(manifest.contributes.menus).toStrictEqual([
      { location: 'titleBar.commands', command: 'tree.open', group: 'navigate', order: 10 },
    ])
  })

  it('declares the command the title bar shows with the title and the icon of a folder tree', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'tree.open')).toMatchObject({
      title: 'Open file tree',
      icon: 'folder-tree',
    })
  })
})

describe('the menu of a row', () => {
  it('declares the New submenu and places renaming in the menu of a row', () => {
    expect(getMenuEntries()).toStrictEqual([
      { slot: 'fileTree.context', submenu: 'fileTree.new', title: 'New', group: '1-new' },
      { slot: 'fileTree.new', command: 'tree.newNote', group: '0-tree', order: 10 },
      { slot: 'fileTree.new', command: 'tree.newFolder', group: '0-tree', order: 20 },
      { slot: 'fileTree.context', command: 'tree.rename', group: '2-edit', order: 10 },
    ])
  })

  it('declares the commands the menu runs, over any row', () => {
    expect(getCommand('tree.rename')).toStrictEqual({
      id: 'tree.rename',
      title: 'Rename',
      icon: 'pencil',
    })
    expect(getCommand('tree.newNote')).toStrictEqual({
      id: 'tree.newNote',
      title: 'New note',
      icon: 'file-plus',
    })
    expect(getCommand('tree.newFolder')).toStrictEqual({
      id: 'tree.newFolder',
      title: 'New folder',
      icon: 'folder-plus',
    })
  })
})

describe('the readme', () => {
  it('names the title bar button that opens the file tree', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
