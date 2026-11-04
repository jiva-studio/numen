/**
 * Which commands the plugins declare the palette runs in a backend, and what the window tells the
 * plugin host about every declared command.
 */
import { describe, expect, it } from 'vitest'
import type { DeclaredCommand } from '@/entities/plugin'
import { useDeclaredCommands } from './declaredCommands'

const base: Omit<DeclaredCommand, 'pluginId' | 'id' | 'title' | 'hasBackend' | 'hasFrontend'> = {
  isListedByMenu: false,
  isSuspended: false,
  contentTypes: [],
  extensions: [],
  arguments: [],
}

const createDeck: DeclaredCommand = {
  ...base,
  pluginId: 'md.cards',
  id: 'deck.create',
  title: 'Create deck',
  hasBackend: true,
  hasFrontend: true,
}
const openPreset: DeclaredCommand = {
  ...base,
  pluginId: 'md.cards',
  id: 'preset.open',
  title: 'Open preset',
  scope: 'note',
  shortcut: 'mod+shift+o',
  isNoteDestination: false,
  hasBackend: true,
  hasFrontend: true,
}
const openTree: DeclaredCommand = {
  ...base,
  pluginId: 'md.tree',
  id: 'tree.open',
  title: 'Open file tree',
  icon: 'folder-tree',
  hasBackend: false,
  hasFrontend: true,
}

async function load(hasHandler: (id: string) => boolean) {
  const declared = useDeclaredCommands({
    list: async () => [createDeck, openPreset, openTree],
    hasHandler,
  })
  await declared.load()
  return declared
}

describe('the declared commands the palette runs in a backend', () => {
  it('are those of plugins that have a backend and that no frontend handler claims', async () => {
    const declared = await load((id) => id === 'md.cards:preset.open')

    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      'md.cards:deck.create',
    ])
  })

  it('leave out the commands of a plugin without a backend', async () => {
    const declared = await load(() => false)

    expect(declared.paletteCommands.value.map((one) => one.id)).not.toContain('md.tree:tree.open')
  })
})

describe('the declarations for the plugin host', () => {
  it('carry every declared command with its title, icon, scope, shortcut and the parts of its plugin', async () => {
    const declared = await load(() => false)

    expect(declared.declarations.value).toStrictEqual([
      {
        pluginId: 'md.cards',
        id: 'deck.create',
        title: 'Create deck',
        hasBackend: true,
        hasFrontend: true,
      },
      {
        pluginId: 'md.cards',
        id: 'preset.open',
        title: 'Open preset',
        scope: 'note',
        shortcut: 'mod+shift+o',
        hasBackend: true,
        hasFrontend: true,
      },
      {
        pluginId: 'md.tree',
        id: 'tree.open',
        title: 'Open file tree',
        icon: 'folder-tree',
        hasBackend: false,
        hasFrontend: true,
      },
    ])
  })
})
