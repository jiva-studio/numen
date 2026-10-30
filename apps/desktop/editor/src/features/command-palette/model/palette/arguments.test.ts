/**
 * A command that requests its arguments one step each: what each step renders,
 * what it continues with, and the values the invocation carries.
 */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { runSupport } from './runSupport'
import { useCommandPalette } from './palette'
import type { Command, CommandTarget } from '../../types'
import { WORDS as words } from '@/shared/words'

/** Throws the rejection reason of a detached promise, so the test run reports it as an unhandled rejection. */
function rethrow(error: unknown): never {
  throw error
}

const front: CommandTarget = {
  tab: 'tab',
  kind: 'note',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: 'physics/Ontology.md',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
}

const createSample: Command = {
  id: 'create-sample',
  text: 'Create a sample',
  group: 'window',
  needs: 'arguments',
  prompts: [
    { name: 'title', title: 'Title', isRequired: true },
    { name: 'folder', title: 'Folder', isRequired: false },
    { name: 'form', title: 'Form', isRequired: false },
  ],
  isAvailable: () => true,
}

const createPalette = () => {
  const commands = useCommandPalette({
    core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    at: () => front,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: vi.fn() },
    runs: runSupport(),
    getExtraCommands: () => [createSample],
  })
  commands.setOpen(true)
  return commands
}

describe('a command that asks for its arguments', () => {
  it('asks for the first in a step named after it', () => {
    const commands = createPalette()

    expect(commands.startCommand('create-sample', front)).toBeNull()

    expect(commands.crumb.value).toBe('Create a sample')
    expect(commands.placeholder.value).toBe('Title')
    expect(commands.groups.value.map((group) => group.id)).toStrictEqual(['arguments'])
    expect(commands.groups.value[0]?.items).toStrictEqual([])
  })

  it('does not go on from a required argument with nothing typed', () => {
    const commands = createPalette()
    commands.startCommand('create-sample', front)

    expect(commands.chooseItem('name', 'name')).toBeNull()
    expect(commands.placeholder.value).toBe('Title')
  })

  it('goes on to the next argument with what was typed, and offers to leave an optional one out', () => {
    const commands = createPalette()
    commands.startCommand('create-sample', front)
    commands.setTyped('  Verbs ').catch(rethrow)

    expect(commands.groups.value[0]?.items[0]?.title).toBe('Title: “Verbs”')
    expect(commands.chooseItem('name', 'name')).toBeNull()

    expect(commands.placeholder.value).toBe('Folder')
    expect(commands.typed.value).toBe('')
    expect(commands.groups.value[0]?.items[0]?.title).toBe(`${words.leavesOut} Folder`)
  })

  it('carries every value typed, by name, and leaves out the optional ones left blank', () => {
    const commands = createPalette()
    commands.startCommand('create-sample', front)
    commands.setTyped('Verbs').catch(rethrow)
    commands.chooseItem('name', 'name')
    commands.chooseItem('name', 'name')
    commands.setTyped('Spanish').catch(rethrow)

    const invocation = commands.chooseItem('name', 'name')

    expect(invocation?.id).toBe('create-sample')
    expect(invocation?.file).toBe('physics/Ontology.md')
    expect(invocation?.values).toStrictEqual({ title: 'Verbs', form: 'Spanish' })
  })

  it('goes back to the argument before', () => {
    const commands = createPalette()
    commands.startCommand('create-sample', front)
    commands.setTyped('Verbs').catch(rethrow)
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
      lists: { getStepGroups: () => [], previewItem: vi.fn() },
      runs: runSupport(),
      getExtraCommands: () => extra.value,
    })
    commands.setOpen(true)
    commands.setTyped('a sample').catch(rethrow)
    expect(commands.groups.value).toStrictEqual([])

    extra.value = [createSample]

    expect(
      commands.groups.value.flatMap((group) => group.items.map((item) => item.id)),
    ).toStrictEqual(['create-sample'])

    expect(commands.startCommand('create-sample', front)).toBeNull()
    expect(commands.placeholder.value).toBe('Title')
  })
})
