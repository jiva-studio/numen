/**
 * A command that asks for its arguments one step each: what each step draws,
 * what it goes on with, and the values the invocation carries.
 */
import { describe, expect, it } from 'vitest'
import { shallowRef } from 'vue'
import { runSupport } from './runSupport'
import { useCommandPalette } from './palette'
import type { Command, CommandTarget } from '../../types'
import { WORDS as words } from '@/shared/words'

const front: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: 'physics/Ontology.md',
  source: null,
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const createDeck: Command = {
  id: 'create-deck',
  text: 'Create a deck',
  group: 'window',
  needs: 'arguments',
  prompts: [
    { name: 'title', title: 'Title', isRequired: true },
    { name: 'folder', title: 'Folder', isRequired: false },
    { name: 'stencil', title: 'Stencil', isRequired: false },
  ],
  isOffered: () => true,
}

const createPalette = () => {
  const commands = useCommandPalette({
    core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    at: () => front,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: () => {} },
    runs: runSupport(),
    getExtraCommands: () => [createDeck],
  })
  commands.setOpen(true)
  return commands
}

describe('a command that asks for its arguments', () => {
  it('asks for the first in a step named after it', () => {
    const commands = createPalette()

    expect(commands.startCommand('create-deck', front)).toBeNull()

    expect(commands.crumb.value).toBe('Create a deck')
    expect(commands.placeholder.value).toBe('Title')
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['arguments'])
    expect(commands.groups.value[0]?.items).toStrictEqual([])
  })

  it('does not go on from a required argument with nothing typed', () => {
    const commands = createPalette()
    commands.startCommand('create-deck', front)

    expect(commands.chooseItem('name', 'name')).toBeNull()
    expect(commands.placeholder.value).toBe('Title')
  })

  it('goes on to the next argument with what was typed, and offers to leave an optional one out', () => {
    const commands = createPalette()
    commands.startCommand('create-deck', front)
    void commands.setTyped('  Verbs ')

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Title: “Verbs”')
    expect(commands.chooseItem('name', 'name')).toBeNull()

    expect(commands.placeholder.value).toBe('Folder')
    expect(commands.typed.value).toBe('')
    expect(commands.groups.value[0]?.items[0]?.title).toBe(`${words.leavesOut} Folder`)
  })

  it('carries every value typed, by name, and leaves out the optional ones left blank', () => {
    const commands = createPalette()
    commands.startCommand('create-deck', front)
    void commands.setTyped('Verbs')
    commands.chooseItem('name', 'name')
    commands.chooseItem('name', 'name')
    void commands.setTyped('Spanish')

    const invocation = commands.chooseItem('name', 'name')

    expect(invocation?.id).toBe('create-deck')
    expect(invocation?.file).toBe('physics/Ontology.md')
    expect(invocation?.values).toStrictEqual({ title: 'Verbs', stencil: 'Spanish' })
  })

  it('goes back to the argument before', () => {
    const commands = createPalette()
    commands.startCommand('create-deck', front)
    void commands.setTyped('Verbs')
    commands.chooseItem('name', 'name')

    expect(commands.goBack()).toBe(false)

    expect(commands.placeholder.value).toBe('Title')
  })
})

describe('a command that arrives after the palette is made', () => {
  it('is offered from then on', () => {
    const extra = shallowRef<readonly Command[]>([])
    const commands = useCommandPalette({
      core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
      words,
      at: () => front,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: () => {} },
      runs: runSupport(),
      getExtraCommands: () => extra.value,
    })
    commands.setOpen(true)
    void commands.setTyped('a deck')
    expect(commands.groups.value).toStrictEqual([])

    extra.value = [createDeck]

    expect(
      commands.groups.value.flatMap((group) => group.items.map((item) => item.id)),
    ).toStrictEqual(['create-deck'])

    expect(commands.startCommand('create-deck', front)).toBeNull()
    expect(commands.placeholder.value).toBe('Title')
  })
})
