/**
 * A command that requests its arguments one step each: what each step renders,
 * what it continues with, and the values the invocation carries.
 */
import { describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { runSupport } from './runSupport'
import { createCommandPalette, type Commands } from './palette'
import type { Command } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'
import { WORDS as windowWords } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../../words'

const words = { ...windowWords, ...PALETTE_WORDS }

function rethrow(error: unknown): never {
  throw error
}

function chooseRow(commands: Commands) {
  const row = commands.view.groups.value.flatMap((group) => group.items)[0]
  return commands.palette.chooseItem(row?.id ?? '', row?.actions?.[0]?.id ?? '')
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
  input: 'arguments',
  prompts: [
    { name: 'title', title: 'Title', isRequired: true },
    { name: 'folder', title: 'Folder', isRequired: false },
    { name: 'form', title: 'Form', isRequired: false },
  ],
  isAvailable: () => true,
}

const createPalette = () => {
  const commands = createCommandPalette({
    core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
    words,
    at: () => front,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: vi.fn() },
    runs: runSupport(),
    getCommands: () => [createSample],
  })
  commands.palette.setOpen(true)
  return commands
}

describe('a command that asks for its arguments', () => {
  it('asks for the first in a step named after it', () => {
    const commands = createPalette()

    expect(commands.palette.startCommand('create-sample', front)).toBeNull()

    expect(commands.navigation.crumb.value).toBe('Create a sample')
    expect(commands.navigation.placeholder.value).toBe('Title')
    expect(commands.view.groups.value.map((group) => group.id)).toStrictEqual(['arguments'])
    expect(commands.view.groups.value[0]?.items).toStrictEqual([])
  })

  it('does not go on from a required argument with nothing typed', () => {
    const commands = createPalette()
    commands.palette.startCommand('create-sample', front)

    expect(chooseRow(commands)).toBeNull()
    expect(commands.navigation.placeholder.value).toBe('Title')
  })

  it('goes on to the next argument with what was typed, and offers to leave an optional one out', () => {
    const commands = createPalette()
    commands.palette.startCommand('create-sample', front)
    commands.palette.setTyped('  Verbs ').catch(rethrow)

    expect(commands.view.groups.value[0]?.items[0]?.title).toBe('Title: “Verbs”')
    expect(chooseRow(commands)).toBeNull()

    expect(commands.navigation.placeholder.value).toBe('Folder')
    expect(commands.palette.typed.value).toBe('')
    expect(commands.view.groups.value[0]?.items[0]?.title).toBe(`${words.leavesOut} Folder`)
  })

  it('carries every value typed, by name, and leaves out the optional ones left blank', () => {
    const commands = createPalette()
    commands.palette.startCommand('create-sample', front)
    commands.palette.setTyped('Verbs').catch(rethrow)
    chooseRow(commands)
    chooseRow(commands)
    commands.palette.setTyped('Spanish').catch(rethrow)

    const invocation = chooseRow(commands)

    expect(invocation?.id).toBe('create-sample')
    expect(invocation?.target.file).toBe('physics/Ontology.md')
    expect(invocation?.values).toStrictEqual({ title: 'Verbs', form: 'Spanish' })
  })

  it('goes back to the argument before', () => {
    const commands = createPalette()
    commands.palette.startCommand('create-sample', front)
    commands.palette.setTyped('Verbs').catch(rethrow)
    chooseRow(commands)

    expect(commands.navigation.goBack()).toBe(false)

    expect(commands.navigation.placeholder.value).toBe('Title')
  })
})

describe('a command that arrives after the palette is made', () => {
  it('is offered from then on', () => {
    const extra = shallowRef<readonly Command[]>([])
    const commands = createCommandPalette({
      core: { names: async () => [], vaults: async () => ({ vaults: [], showing: '' }) },
      words,
      at: () => front,
      knows: { getTitle: () => '', getTabAt: () => null },
      lists: { getStepGroups: () => [], previewItem: vi.fn() },
      runs: runSupport(),
      getCommands: () => extra.value,
    })
    commands.palette.setOpen(true)
    commands.palette.setTyped('a sample').catch(rethrow)
    expect(commands.view.groups.value).toStrictEqual([])

    extra.value = [createSample]

    expect(
      commands.view.groups.value.flatMap((group) => group.items.map((item) => item.id)),
    ).toStrictEqual(['create-sample'])

    expect(commands.palette.startCommand('create-sample', front)).toBeNull()
    expect(commands.navigation.placeholder.value).toBe('Title')
  })
})
