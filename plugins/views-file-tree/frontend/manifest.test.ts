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
    slots?: readonly { slot: string; command?: string; view?: string; group?: string; order?: number }[]
  }
}

describe('the manifest', () => {
  it('declares vault:write, which the files the tree moves and makes need', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read', 'vault:write'])
  })

  it('places opening the file tree first among the navigation entries of the title bar', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'titleBar.commands', command: 'tree.open', group: 'navigate', order: 10 },
    ])
  })

  it('declares the command the title bar shows with the title and the icon of a folder tree', () => {
    expect(manifest.contributes.commands.find((one) => one.id === 'tree.open')).toMatchObject({
      title: 'Open file tree',
      icon: 'folder-tree',
    })
  })
})

describe('the readme', () => {
  it('names the title bar button that opens the file tree', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
