import { describe, expect, it } from 'vitest'
import type { PluginHost } from '@numen/plugin-host'
import type { VaultCore } from '../vault'
import { usePluginCommands } from './usePluginCommands'

describe('the palette entry of a command a plugin declares with a Shift shortcut', () => {
  it('is drawn with the keystroke holding Shift that the manifest names', () => {
    const host = {
      getPaletteCommands: () => [
        { id: 'md.agent:chat.open', title: 'Open agent chat', shortcut: 'mod+shift+a' },
        { id: 'md.graph:graph.open', title: 'Open plex graph', shortcut: 'mod+shift+x' },
      ],
    } as unknown as PluginHost

    const entries = usePluginCommands({} as VaultCore, host).getPaletteEntries()

    expect(entries.map((one) => [one.keys?.letter, one.keys?.icons.includes('shift')])).toEqual([
      ['A', true],
      ['X', true],
    ])
  })
})
