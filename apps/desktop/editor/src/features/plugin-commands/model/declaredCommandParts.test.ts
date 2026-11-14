/**
 * Which commands the plugins declare the palette runs in a backend, and what the window tells the
 * plugin host about every declared command.
 */
import { describe, expect, it } from 'vitest'
import type { DeclaredCommand } from '@/entities/plugin'
import { useDeclaredCommands } from './declaredCommands'
import { UNANSWERED_RUNS } from '@/testing/declaredCommands'

const base: Omit<DeclaredCommand, 'pluginId' | 'id' | 'title'> = {
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
  when: 'target.isNote',
}
const openTree: DeclaredCommand = {
  ...base,
  pluginId: 'md.tree',
  id: 'tree.open',
  title: 'Open file tree',
  icon: 'folder-tree',
}

async function load(
  isListed: (id: string) => boolean,
  hasBackend: (pluginId: string) => boolean = () => true,
) {
  const declared = useDeclaredCommands({ ...UNANSWERED_RUNS,
    list: async () => ({ commands: [createDeck, openPreset, openTree], slots: [] }),
    isListed,
    hasBackend,
  })
  await declared.load()
  return declared
}

describe('the declared commands the palette runs in a backend', () => {
  it('are those the plugin host does not list', async () => {
    const declared = await load((id) => id === 'md.cards:preset.open' || id === 'md.tree:tree.open')

    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      'md.cards:deck.create',
    ])
  })

  it('leave out the commands of a plugin that has no backend', async () => {
    const declared = await load(
      () => false,
      (pluginId) => pluginId !== 'md.tree',
    )

    expect(declared.paletteCommands.value.map((one) => one.id)).toStrictEqual([
      'md.cards:deck.create',
      'md.cards:preset.open',
    ])
  })

  it('leave out the commands the plugin host lists', async () => {
    const declared = await load((id) => id === 'md.tree:tree.open')

    expect(declared.paletteCommands.value.map((one) => one.id)).not.toContain('md.tree:tree.open')
  })
})

describe('the declarations for the plugin host', () => {
  it('carry every declared command with its title, icon and condition', async () => {
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
        when: 'target.isNote',
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

describe('the handlers for the plugin host', () => {
  it('are the handlers of shared commands the core lists, undefined before the first list', async () => {
    const handler = {
      pluginId: 'md.pages',
      command: 'app:file.createText',
      title: 'Extract text layer',
      hasItems: false,
    }
    const declared = useDeclaredCommands({ ...UNANSWERED_RUNS,
      list: async () => ({ commands: [], slots: [], handlers: [handler] }),
    })
    expect(declared.contributions.handlers.value).toBeUndefined()

    await declared.load()

    expect(declared.contributions.handlers.value).toStrictEqual([handler])
  })
})
