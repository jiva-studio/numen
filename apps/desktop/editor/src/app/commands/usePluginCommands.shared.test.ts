/**
 * The shared commands in the palette: one entry for each the enabled plugins handle, whose pick
 * list is its handlers, run over the target with the one picked.
 */
import { describe, expect, it, vi } from 'vitest'
import type { HandlerDeclaration, PluginHost } from '@numen/plugin-host'
import type { CommandTarget } from '@/features/command-palette'
import type { VaultCore } from '../vault'
import { usePluginCommands } from './usePluginCommands'

const HANDLER: HandlerDeclaration = {
  pluginId: 'md.pages',
  command: 'app:file.createText',
  title: 'Extract text layer',
  hasItems: false,
}

const createHost = () =>
  ({
    hasCommandItems: () => true,
    getPaletteCommands: () => [],
    executeCommand: vi.fn(async () => true),
  }) as unknown as PluginHost

describe('a shared command in the palette', () => {
  it('is an entry with a pick list while a plugin handles it, and none otherwise', () => {
    const handled = usePluginCommands({} as VaultCore, createHost(), undefined, () => [HANDLER])
    const unhandled = usePluginCommands({} as VaultCore, createHost(), undefined, () => [])

    expect(handled.getPaletteEntries()).toMatchObject([
      { id: 'app:file.createText', text: 'Create text', input: 'items', group: 'note' },
    ])
    expect(unhandled.getPaletteEntries()).toStrictEqual([])
  })

  it('is available over a file and not over the window alone', () => {
    const [entry] = usePluginCommands({} as VaultCore, createHost(), undefined, () => [
      HANDLER,
    ]).getPaletteEntries()
    const target = { path: 'scan.pdf', file: 'scan.pdf', isReady: true } as CommandTarget

    expect(entry?.isAvailable?.(target)).toBe(true)
    expect(entry?.isAvailable?.({ ...target, path: '', file: '' })).toBe(false)
  })
})
