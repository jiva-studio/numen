import { describe, expect, it } from 'vitest'
import type { PluginHost } from '@numen/plugin-loader'
import { createCommandRegistry } from '@/shared/model/command'
import { getFileKinds } from '@/entities/file'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

describe('the palette entry of a command a plugin declares with a Shift shortcut', () => {
  it('is drawn with the keystroke holding Shift that the manifest names', () => {
    const shortcuts: Record<string, string> = {
      'md.agent:chat.open': 'mod+shift+a',
      'md.graph:graph.open': 'mod+shift+x',
    }
    const host = {
      commands: {
        hasCommandItems: () => false,
        getPaletteCommands: () => [
          { id: 'md.agent:chat.open', title: 'Open agent chat' },
          { id: 'md.graph:graph.open', title: 'Open plex graph' },
        ],
      },
      keybindings: { getKeyFor: (id: string) => shortcuts[id] },
    } as unknown as PluginHost

    const entries = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      host,
    ).getPaletteEntries()

    expect(entries.map((one) => [one.keys?.letter, one.keys?.icons.includes('shift')])).toEqual([
      ['A', true],
      ['X', true],
    ])
  })
})
