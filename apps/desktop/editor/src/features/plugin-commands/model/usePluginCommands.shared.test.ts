/**
 * The shared commands in the palette: one entry for each the enabled plugins handle, whose pick
 * list is its handlers, run over the target with the one picked.
 */
import { createCommandRegistry } from '@/shared/model/command'
import { describe, expect, it, vi } from 'vitest'
import type { HandlerDeclaration, PluginHost } from '@numen/plugin-loader'
import type { CommandTarget, RunSupport } from '@/shared/model/command'
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
