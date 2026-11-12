import { describe, expect, it } from 'vitest'
import type { PluginHost } from '@numen/plugin-loader'
import { getNamedIcon } from '@/shared/ui/icons'
import { createCommandRegistry } from '@/shared/model/command'
import { getFileKinds } from '@/entities/file'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

describe('the palette entry of a command a plugin declares', () => {
  it('is drawn with the icon the manifest names', () => {
    const host = {
      commands: {
        hasCommandItems: () => false,
        getPaletteCommands: () => [
          { id: 'md.tree:tree.open', title: 'Open file tree', icon: 'folder-tree' },
        ],
      },
      keybindings: { getKeyFor: () => undefined },
    } as unknown as PluginHost

    const [entry] = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ fileKinds: getFileKinds }),
      host,
    ).getPaletteEntries()

    expect(entry?.icon).toBe(getNamedIcon('folder-tree'))
  })
})
