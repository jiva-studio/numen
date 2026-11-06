/**
 * The entries the plugin's manifest places in the New submenu of the file tree: the commands that
 * create a deck, a card model and a preset.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  contributes: {
    commands: readonly Record<string, unknown>[]
    slots?: readonly Record<string, unknown>[]
  }
}

describe('the New submenu of the file tree', () => {
  it('holds the commands that create a deck, a card model and a preset, over a folder', () => {
    const overFolder = { group: '1-cards', when: 'target.isFolder' }
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'fileTree.new'),
    ).toStrictEqual([
      { slot: 'fileTree.new', command: 'deck.create', order: 10, ...overFolder },
      { slot: 'fileTree.new', command: 'cardModel.create', order: 20, ...overFolder },
      { slot: 'fileTree.new', command: 'preset.create', order: 30, ...overFolder },
    ])
  })

  it('declares one command for each, the one the palette lists', () => {
    const ids = manifest.contributes.commands.map((one) => one.id)

    expect(ids).toContain('deck.create')
    expect(ids).not.toContain('deck.new')
    expect(ids).not.toContain('cardModel.new')
    expect(ids).not.toContain('preset.new')
  })
})
