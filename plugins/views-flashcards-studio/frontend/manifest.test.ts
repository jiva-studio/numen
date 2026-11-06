/**
 * The entries the plugin's manifest places in the New submenu of the file tree, and the commands
 * they run.
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

const getCommand = (id: string) => manifest.contributes.commands.find((one) => one.id === id)

describe('the New submenu of the file tree', () => {
  it('holds making a deck, a card model and a preset', () => {
    expect(
      (manifest.contributes.slots ?? []).filter((one) => one.slot === 'fileTree.new'),
    ).toStrictEqual([
      { slot: 'fileTree.new', command: 'deck.new', group: '1-cards', order: 10 },
      { slot: 'fileTree.new', command: 'cardModel.new', group: '1-cards', order: 20 },
      { slot: 'fileTree.new', command: 'preset.new', group: '1-cards', order: 30 },
    ])
  })

  it('declares the commands it runs, over any row and with no argument', () => {
    expect(getCommand('deck.new')).toStrictEqual({
      id: 'deck.new',
      title: 'New deck',
      icon: 'layers',
    })
    expect(getCommand('cardModel.new')).toStrictEqual({
      id: 'cardModel.new',
      title: 'New card model',
      icon: 'layout-template',
    })
    expect(getCommand('preset.new')).toStrictEqual({
      id: 'preset.new',
      title: 'New preset',
      icon: 'gauge',
    })
  })
})
