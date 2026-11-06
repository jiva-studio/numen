import { describe, expect, it } from 'vitest'
import type { PluginHost } from '@numen/plugin-host'
import { getNamedIcon } from '@/shared/icons'
import type { VaultCore } from '../vault'
import { usePluginCommands } from './usePluginCommands'

describe('the palette entry of a command a plugin declares', () => {
  it('is drawn with the icon the manifest names', () => {
    const host = {
      hasCommandItems: () => false,
      getPaletteCommands: () => [
        { id: 'md.tree:tree.open', title: 'Open file tree', icon: 'folder-tree' },
      ],
    } as unknown as PluginHost

    const [entry] = usePluginCommands({} as VaultCore, host).getPaletteEntries()

    expect(entry?.icon).toBe(getNamedIcon('folder-tree'))
  })
})
