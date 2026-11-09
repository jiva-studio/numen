/**
 * The shared commands in the palette: one entry for each the enabled plugins handle, whose pick
 * list is its handlers, run over the target with the one picked.
 */
import { describe, expect, it, vi } from 'vitest'
import type { HandlerDeclaration, PluginHost } from '@numen/plugin-loader'
import {
  commands,
  type AnswerWords,
  type CommandDeps,
  type CommandInvocation,
  type CommandTarget,
  type RunSupport,
} from '@/features/command-palette'
import type { VaultCore } from '../vault'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

const HANDLER: HandlerDeclaration = {
  pluginId: 'md.pages',
  command: 'app:file.createText',
  title: 'Extract text layer',
  hasItems: false,
}

const RUNS: RunSupport = { canRun: () => true, cannotRun: () => undefined }

const createHost = () =>
  ({
    hasCommandItems: () => true,
    getPaletteCommands: () => [],
    executeCommand: vi.fn(async () => true),
  }) as unknown as PluginHost

describe('a shared command in the palette', () => {
  it('is an entry with a pick list while a plugin handles it, and none otherwise', () => {
    const handled = usePluginCommands(
      createCommandNoteReader({} as VaultCore),
      createHost(),
      undefined,
      () => [HANDLER],
    )
    const unhandled = usePluginCommands(
      createCommandNoteReader({} as VaultCore),
      createHost(),
      undefined,
      () => [],
    )

    expect(handled.getPaletteEntries()).toMatchObject([
      { id: 'app:file.createText', text: 'Create text', input: 'items', group: 'note' },
    ])
    expect(unhandled.getPaletteEntries()).toStrictEqual([])
  })

  it('is available over a file and not over the window alone', () => {
    const [entry] = usePluginCommands(
      createCommandNoteReader({} as VaultCore),
      createHost(),
      undefined,
      () => [HANDLER],
    ).getPaletteEntries()
    const target = { path: 'scan.pdf', file: 'scan.pdf', isReady: true } as CommandTarget

    expect(entry?.isAvailable(target, RUNS)).toBe(true)
    expect(entry?.isAvailable({ ...target, path: '', file: '' }, RUNS)).toBe(false)
  })
})

describe('a shared command that asks for a value', () => {
  const importer = { ...HANDLER, command: 'app:link.import', title: 'Import link' }

  it('asks for its required arguments as a command of a manifest does', () => {
    const [entry] = usePluginCommands(
      createCommandNoteReader({} as VaultCore),
      createHost(),
      undefined,
      () => [importer],
    ).getPaletteEntries()

    expect(entry).toMatchObject({
      id: 'app:link.import',
      text: 'Import link',
      input: 'arguments',
      prompts: [{ name: 'address', title: 'Web address', isRequired: true }],
    })
    const window = {
      path: '',
      file: '',
      tab: '',
      isReady: true,
      vault: { id: 'v', name: 'Harbor' },
    }

    expect(entry?.isAvailable(window as CommandTarget, RUNS)).toBe(true)
  })

  it('runs its handler with the values given', async () => {
    const host = createHost()
    usePluginCommands(
      createCommandNoteReader({ fileKinds: async () => new Map() } as unknown as VaultCore),
      host,
      undefined,
      () => [importer],
    )
    const run = commands.get('app:link.import')

    await run?.(
      {
        id: 'app:link.import',
        path: '',
        name: '',
        title: '',
        others: [],
        tab: '',
        values: { address: 'https://example.com/tides' },
      } as unknown as CommandInvocation,
      {} as CommandDeps,
      {} as AnswerWords,
    )

    expect(host.executeCommand).toHaveBeenCalledWith(
      'app:link.import',
      expect.objectContaining({ path: '' }),
      { values: { address: 'https://example.com/tides' } },
    )
  })
})
