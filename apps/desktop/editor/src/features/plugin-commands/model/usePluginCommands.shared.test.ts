/**
 * The shared commands in the palette: one entry for each the enabled plugins handle, whose pick
 * list is its handlers, run over the target with the one picked.
 */
import { createCommandRegistry, type CommandDeps } from '@/shared/model/command'
import { describe, expect, it, vi } from 'vitest'
import type { HandlerDeclaration, PluginHost } from '@numen/plugin-loader'
import type { CommandInvocation, CommandTarget, RunSupport } from '@/shared/model/command'
import { getFileKinds } from '@/entities/file'
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
    commands: {
      hasCommandItems: () => true,
      getPaletteCommands: () => [],
      executeCommand: vi.fn(async () => true),
    },
  }) as unknown as PluginHost

describe('a shared command in the palette', () => {
  it('is an entry with a pick list while a plugin handles it, and none otherwise', () => {
    const handled = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [HANDLER] },
    )
    const unhandled = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [] },
    )

    expect(handled.getPaletteEntries()).toMatchObject([
      { id: 'app:file.createText', text: 'Create text', input: 'items', group: 'note' },
    ])
    expect(unhandled.getPaletteEntries()).toStrictEqual([])
  })

  it('is available over a file and not over the window alone', () => {
    const [entry] = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [HANDLER] },
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
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      createHost(),
      { getHandlers: () => [importer] },
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
    const registry = createCommandRegistry()
    usePluginCommands(
      registry,
      createCommandNoteReader({ fileKinds: async () => new Map() }),
      host,
      { getHandlers: () => [importer] },
    )
    const run = registry.get('app:link.import')

    await run?.(
      {
        id: 'app:link.import',
        target: {
          path: '',
          title: '',
          others: [],
          tab: '',
        },
        name: '',
        values: { address: 'https://example.com/tides' },
      } as unknown as CommandInvocation,
      {} as CommandDeps,
    )

    expect(host.commands.executeCommand).toHaveBeenCalledWith(
      'app:link.import',
      expect.objectContaining({ path: '' }),
      { values: { address: 'https://example.com/tides' } },
    )
  })
})
