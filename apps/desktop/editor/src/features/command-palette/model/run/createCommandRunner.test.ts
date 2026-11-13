/** How a command of the window is run from the palette and the keys, and to its end for a plugin. */
import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { type Command } from '@/entities/command'
import {
  createCommandRegistry,
  type CommandInvocation,
  type CommandTarget,
} from '@/shared/model/command'
import { WORDS } from '@/shared/ui/words'
import { PALETTE_WORDS } from '../../words'
import { createCommandPalette } from '../palette/palette'
import { runSupport } from '../palette/runSupport'
import { createCommandDeps } from './createCommandDeps'
import { createCommandRunner } from './createCommandRunner'

const FAILURE = createLostConnection('the core is gone')
const FAILURE_TEXT = 'Numen is not responding; retrying'

const TARGET: CommandTarget = {
  tab: 'note:1',
  kind: 'note',
  path: 'Tides.md',
  title: 'Tides',
  file: 'Tides.md',
  vault: { id: 'sea', name: 'Sea' },
  isReady: true,
}

const row = (id: string, more: Partial<Command> = {}): Command => ({
  id,
  text: id,
  group: 'note',
  isAvailable: (target) => target.path !== '',
  ...more,
})

const setup = () => {
  const ran: string[] = []
  const messages: [string, string | undefined][] = []
  const runs = runSupport()
  const registry = createCommandRegistry()
  registry.register('app:note.save', (invocation: CommandInvocation) => {
    ran.push(invocation.id)
  })
  registry.register('app:note.export', () => 'exports/Tides.md')
  registry.register('app:note.fail', () => {
    throw FAILURE
  })
  const commands = createCommandPalette({
    core: {
      names: () => Promise.resolve([]),
      vaults: () => Promise.resolve({ vaults: [], showing: '' }),
    },
    words: { ...WORDS, ...PALETTE_WORDS },
    at: () => TARGET,
    knows: { getTitle: () => '', getTabAt: () => null },
    lists: { getStepGroups: () => [], previewItem: () => undefined },
    runs,
    getCommands: () => [
      row('app:note.save'),
      row('app:note.export'),
      row('app:note.fail'),
      row('app:note.create', { input: 'name' }),
    ],
  })
  const pluginCommands = {
    isPluginCommand: (id: string) => id.startsWith('org.example.'),
    run: vi.fn(() => Promise.resolve(true)),
  }
  const closeSearch = vi.fn()
  const commandDeps = createCommandDeps({
    windowTabs: {} as never,
    reached: {} as never,
    settingPalette: { choose: () => Promise.resolve() },
    runs,
    registry,
    writeMessage: (text, severity) => messages.push([text, severity]),
    navigation: { search: () => undefined },
  })
  const runner = createCommandRunner({
    commands,
    closeSearch,
    pluginCommands,
    getCommandDeps: () => commandDeps,
    writeMessage: (text, severity) => messages.push([text, severity]),
  })
  /** The messages texts, without the empty one that clears the last. */
  const texts = () => messages.filter(([text]) => text !== '')
  return { runner, commands, ran, texts, pluginCommands, closeSearch }
}

describe('running a command', () => {
  it('runs the invocation the palette starts', async () => {
    const { runner, ran } = setup()

    runner.runCommand('app:note.save', TARGET)

    await vi.waitFor(() => expect(ran).toStrictEqual(['app:note.save']))
  })

  it('opens the step a command needs, runs nothing and closes the search', () => {
    const { runner, commands, ran, closeSearch } = setup()

    runner.runCommand('app:note.create', TARGET)

    expect(commands.palette.open.value).toBe(true)
    expect(closeSearch).toHaveBeenCalledOnce()
    expect(ran).toStrictEqual([])
  })

  it('writes the objection as an error where the command is not offered over the target', () => {
    const { runner, texts } = setup()

    runner.runCommand('app:note.save', { ...TARGET, path: '' })

    expect(texts()).toStrictEqual([[PALETTE_WORDS.noNote, 'error']])
  })

  it('writes the error of a command that fails', async () => {
    const { runner, texts } = setup()

    runner.runCommand('app:note.fail', TARGET)

    await vi.waitFor(() => expect(texts()).toContainEqual([FAILURE_TEXT, 'error']))
  })

  it('runs a command of a plugin in the plugin host with the values given, and closes the commands', () => {
    const { runner, commands, pluginCommands } = setup()
    commands.palette.setOpen(true)

    runner.runCommand('org.example.tide', TARGET, undefined, { kind: 'plex' })

    expect(commands.palette.open.value).toBe(false)
    expect(pluginCommands.run).toHaveBeenCalledExactlyOnceWith('org.example.tide', TARGET, {
      values: { kind: 'plex' },
    })
  })
})

describe('a run to its end, as a plugin awaits it', () => {
  it('settles with the path of the note the command made', async () => {
    const { runner, texts } = setup()

    expect(await runner.runCommandToEnd('app:note.export', TARGET)).toBe('exports/Tides.md')
    expect(texts()).toStrictEqual([])
  })

  it('settles with an empty path for a command that made none', async () => {
    const { runner } = setup()

    expect(await runner.runCommandToEnd('app:note.save', TARGET)).toBe('')
  })

  it('rejects with the error of the command and writes it nowhere', async () => {
    const { runner, texts } = setup()

    await expect(runner.runCommandToEnd('app:note.fail', TARGET)).rejects.toThrow(FAILURE_TEXT)
    expect(texts()).toStrictEqual([])
  })

  it('rejects with the objection to a command that cannot start', async () => {
    const { runner, texts } = setup()

    await expect(runner.runCommandToEnd('app:note.save', { ...TARGET, path: '' })).rejects.toThrow(
      PALETTE_WORDS.noNote,
    )
    expect(texts()).toStrictEqual([])
  })

  it('has the plugin host await a command of a plugin', async () => {
    const { runner, pluginCommands } = setup()

    expect(await runner.runCommandToEnd('org.example.tide', TARGET)).toBe('')
    expect(pluginCommands.run).toHaveBeenCalledWith('org.example.tide', TARGET, {
      isAwaited: true,
    })
  })
})
