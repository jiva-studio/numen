/**
 * Which commands the plugins declare the palette runs in a backend, and what the window tells the
 * plugin host about every declared command.
 */
import { describe, expect, it } from 'vitest'
import type { DeclaredCommand } from '@/entities/plugin'
import { useDeclaredCommands } from './declaredCommands'

const base: Omit<DeclaredCommand, 'pluginId' | 'id' | 'title'> = {
  contentTypes: [],
  extensions: [],
  arguments: [],
}

const createDeck: DeclaredCommand = {
  ...base,
  pluginId: 'md.cards',
  id: 'deck.create',
  title: 'Create deck',
}
const openPreset: DeclaredCommand = {
  ...base,
  pluginId: 'md.cards',
  id: 'preset.open',
  title: 'Open preset',
  scope: 'note',
  shortcut: 'mod+shift+o',
  isNoteDestination: false,
}
const openTree: DeclaredCommand = {
  ...base,
  pluginId: 'md.tree',
  id: 'tree.open',
  title: 'Open file tree',
  icon: 'folder-tree',
}

async function load(isOffered: (id: string) => boolean) {
  const declared = useDeclaredCommands({
    list: async () => [createDeck, openPreset, openTree],
    isOffered,
  })
  await declared.load()
  return declared
}

describe('the declared commands the palette runs in a backend', () => {
  it('are those the plugin host does not offer', async () => {
    const declared = await load((id) => id === 'md.cards:preset.open' || id === 'md.tree:tree.open')

    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      'md.cards:deck.create',
    ])
  })

  it('leave out the commands the plugin host offers', async () => {
    const declared = await load((id) => id === 'md.tree:tree.open')

    expect(declared.paletteCommands.value.map((one) => one.id)).not.toContain('md.tree:tree.open')
  })
})

describe('the declarations for the plugin host', () => {
  it('carry every declared command with its title, icon, scope and shortcut', async () => {
    const declared = await load(() => false)

    expect(declared.declarations.value).toStrictEqual([
      {
        pluginId: 'md.cards',
        id: 'deck.create',
        title: 'Create deck',
      },
      {
        pluginId: 'md.cards',
        id: 'preset.open',
        title: 'Open preset',
        scope: 'note',
        shortcut: 'mod+shift+o',
      },
      {
        pluginId: 'md.tree',
        id: 'tree.open',
        title: 'Open file tree',
        icon: 'folder-tree',
      },
    ])
  })
})
