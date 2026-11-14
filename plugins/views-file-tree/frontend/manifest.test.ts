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
  settings?: readonly {
    key: string
    type: string
    title: string
    options?: readonly string[]
    default?: unknown
  }[]
  contributes: {
    commands: readonly { id: string; title: string; icon: string }[]
    keybindings?: readonly { key: string; command: string }[]
    slots?: readonly Record<string, unknown>[]
  }
}

const getMenuEntries = () =>
  (manifest.contributes.slots ?? []).filter((one) => String(one.slot).startsWith('fileTree.'))

const getCommand = (id: string) => manifest.contributes.commands.find((one) => one.id === id)

describe('the manifest', () => {
  it('declares vault:write, which the files the tree moves and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places opening the file tree first among the navigation entries of the title bar', () => {
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'titleBar.commands'),
    ).toStrictEqual([
      { slot: 'titleBar.commands', command: 'tree.open', group: 'navigate', order: 10 },
    ])
  })

  it('places showing a note in the file tree in the menu of a search result, after showing it in the plex', () => {
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'palette.searchResult'),
    ).toStrictEqual([
      {
        slot: 'palette.searchResult',
        command: 'app:file.reveal',
        values: { kind: 'files' },
        title: 'Show in file tree',
        icon: 'folder-open',
        group: 'navigate',
        order: 30,
        when: 'target.isNote',
      },
    ])
  })

  it('places showing the note in front in the file tree among the commands of the palette over a note', () => {
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'palette.note'),
    ).toStrictEqual([
      {
        slot: 'palette.note',
        command: 'app:file.reveal',
        values: { kind: 'files' },
        title: 'Show in file tree',
        icon: 'folder-open',
        when: 'target.isNote',
      },
    ])
  })

  it('declares the command the title bar shows with the title and the icon of a folder tree', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'tree.open')).toMatchObject({
      title: 'Open file tree',
      icon: 'folder-tree',
    })
  })
})

describe('the settings', () => {
  it('declares a sort setting of name and modified, name by default', () => {
    const setting = manifest.settings?.find((one) => one.key === 'sortBy')

    expect(setting).toMatchObject({
      type: 'enum',
      options: ['name', 'modified'],
      default: 'name',
    })
    expect(setting?.title).toBeTruthy()
  })
})

describe('the menu of a row', () => {
  it('declares the New submenu over a folder and renaming over one row that is not the root', () => {
    expect(getMenuEntries()).toStrictEqual([
      {
        slot: 'fileTree.context',
        submenu: 'fileTree.new',
        title: 'New',
        icon: 'plus',
        group: '1-new',
        when: 'target.isFolder',
      },
      {
        slot: 'fileTree.new',
        command: 'app:note.create',
        group: '0-tree',
        order: 10,
        when: 'target.isFolder',
      },
      {
        slot: 'fileTree.new',
        command: 'app:folder.create',
        group: '0-tree',
        order: 20,
        when: 'target.isFolder',
      },
      {
        slot: 'fileTree.context',
        command: 'tree.renameFile',
        group: '2-edit',
        order: 10,
        when: '!target.isRoot && !target.isMultiple',
      },
    ])
  })

  it('declares the commands the menu runs, over any row', () => {
    expect(getCommand('tree.renameFile')).toStrictEqual({
      id: 'tree.renameFile',
      title: 'Rename',
      icon: 'pencil',
    })
    expect(getCommand('tree.createNote')).toBeUndefined()
    expect(getCommand('tree.createFolder')).toBeUndefined()
  })
})

describe('the shortcut', () => {
  it('opens the file tree on the window key with Shift and E', () => {
    expect(manifest.contributes.keybindings).toContainEqual({
      key: 'mod+shift+e',
      command: 'tree.open',
    })
  })
})

describe('the readme', () => {
  it('names the title bar button that opens the file tree', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
