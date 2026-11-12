/** The runs of one command over a target: with an item, with values, with a name, and for a vault. */
import { describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import {
  COMMAND,
  createCommandRegistry,
  type CommandInvocation,
  type CommandTarget,
} from '@/shared/model/command'
import { runSupport } from '../palette/runSupport'
import { createAppItemRunner } from './createAppItemRunner'
import { createChoiceRunner } from './createChoiceRunner'
import { createCommandDeps } from './createCommandDeps'
import { createInvocationRunner } from './createInvocationRunner'
import { createValuesRunner } from './createValuesRunner'
import { createVaultOpener } from './createVaultOpener'

const TARGET: CommandTarget = {
  tab: 'note:1',
  kind: 'note',
  path: 'Tides.md',
  title: 'Tides',
  file: 'Tides.md',
  vault: { id: 'sea', name: 'Sea' },
  isReady: true,
}

const setup = () => {
  const runs: CommandInvocation[] = []
  const messages: [string, string | undefined][] = []
  const registry = createCommandRegistry()
  const record = (invocation: CommandInvocation) => {
    runs.push(invocation)
  }
  for (const id of ['app:note.move', 'app:file.reveal', 'app:folder.create', COMMAND.openVault]) {
    registry.register(id, record)
  }
  registry.register('app:note.fail', () => {
    throw createLostConnection('the core is gone')
  })
  const deps = createCommandDeps({
    windowTabs: {} as never,
    reached: {} as never,
    settingPalette: { choose: () => Promise.resolve() },
    runs: runSupport(),
    registry,
    writeMessage: (text, severity) => messages.push([text, severity]),
    navigation: { search: () => undefined },
  })
  return { deps, runs, messages }
}

describe('a command run with the item picked for it', () => {
  it('names the item in the invocation and carries the values', async () => {
    const { deps, runs } = setup()

    await createAppItemRunner(deps)('app:note.move', TARGET, 'folder-2', { keep: 'yes' })

    expect(runs).toStrictEqual([
      expect.objectContaining({ id: 'app:note.move', name: 'folder-2', values: { keep: 'yes' } }),
    ])
  })

  it('rejects with the error of the command', async () => {
    const { deps } = setup()

    await expect(createAppItemRunner(deps)('app:note.fail', TARGET, 'x')).rejects.toThrow(
      'Numen is not responding; retrying',
    )
  })
})

describe('a command run with the values of a keybinding', () => {
  it('runs over the target with the values', async () => {
    const { deps, runs } = setup()

    createValuesRunner(deps, vi.fn())('app:file.reveal', TARGET, { kind: 'plex' })

    await vi.waitFor(() =>
      expect(runs).toStrictEqual([
        expect.objectContaining({ id: 'app:file.reveal', values: { kind: 'plex' } }),
      ]),
    )
  })
})

describe('a command run with a name', () => {
  it('runs over the target with the name and settles when it ends', async () => {
    const { deps, runs } = setup()

    await createInvocationRunner(() => deps)('app:folder.create', TARGET, 'Zoology')

    expect(runs).toStrictEqual([
      expect.objectContaining({ id: 'app:folder.create', name: 'Zoology' }),
    ])
  })
})

describe('opening a vault', () => {
  it('runs the command that opens one over the vault', async () => {
    const { deps, runs } = setup()

    await createVaultOpener(deps, () => TARGET)({ id: 'heat', name: 'Heat' })

    expect(runs[0]?.target.vault).toStrictEqual({ id: 'heat', name: 'Heat' })
  })
})

describe('the action of a chosen result', () => {
  it('runs the command over the place it was chosen on, with the text typed', () => {
    const run = vi.fn()

    createChoiceRunner(
      run,
      () => TARGET,
    )({
      command: 'app:file.open',
      input: 'Moons',
      over: { path: 'Moons.md', file: 'Moons.md', title: 'Moons', line: 4 },
    } as never)

    expect(run).toHaveBeenCalledExactlyOnceWith(
      'app:file.open',
      expect.objectContaining({ tab: '', path: 'Moons.md', title: 'Moons', line: 4 }),
      'Moons',
      undefined,
    )
  })

  it('runs the command over the target in front where it names no place', () => {
    const run = vi.fn()

    createChoiceRunner(run, () => TARGET)({ command: 'app:note.save', input: '' } as never)

    expect(run).toHaveBeenCalledExactlyOnceWith('app:note.save', TARGET, undefined, undefined)
  })
})
