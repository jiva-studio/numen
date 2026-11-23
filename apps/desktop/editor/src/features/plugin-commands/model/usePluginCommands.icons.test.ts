import { describe, expect, it } from 'vitest'
import type { PluginHost } from '@numen/plugin-loader'
import { getNamedIcon } from '@/shared/ui/icons'
import { createCommandRegistry } from '@/shared/model/command'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

/** The kinds of files, over a core that answers none. */
const { getFileKinds } = createAnsweredApi({}).vault.file

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
      createCommandNoteReader({ readFileKinds: getFileKinds }),
      host,
    ).getPaletteEntries()

    expect(entry?.icon).toBe(getNamedIcon('folder-tree'))
  })
})

describe('the palette entry of a command a plugin registers with a confirmation declared', () => {
  const answers = {
    keeps: 'Keep',
    retainedNotice: 'Nothing changes',
    action: 'Delete',
    detail: 'From',
  }
  const entryOf = (hasItems: boolean) => {
    const host = {
      commands: {
        hasCommandItems: () => hasItems,
        getPaletteCommands: () => [{ id: 'md.tags:tags.delete', title: 'Delete tag', answers }],
      },
      keybindings: { getKeyFor: () => undefined },
    } as unknown as PluginHost
    return usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ readFileKinds: getFileKinds }),
      host,
    ).getPaletteEntries()[0]
  }

  it('asks it after the item picked where the command has items', () => {
    const entry = entryOf(true)

    expect([entry?.input, entry?.next, entry?.answers]).toStrictEqual([
      'items',
      'confirmation',
      answers,
    ])
  })

  it('asks it first where the command has none', () => {
    const entry = entryOf(false)

    expect([entry?.input, entry?.next, entry?.answers]).toStrictEqual([
      'confirmation',
      undefined,
      answers,
    ])
  })
})

describe('the palette group of a command a plugin declares', () => {
  const groupOf = (when: string) => {
    const host = {
      commands: {
        hasCommandItems: () => false,
        getPaletteCommands: () => [{ id: 'md.tree:tree.open', title: 'Open', when }],
      },
      keybindings: { getKeyFor: () => undefined },
    } as unknown as PluginHost
    return usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ readFileKinds: getFileKinds }),
      host,
    ).getPaletteEntries()[0]?.group
  }

  it('is over the file for a condition on the target, as before its frontend starts', () => {
    expect(groupOf('target.isFolder')).toBe('file')
  })

  it('is over the window for a condition that reads no target', () => {
    expect(groupOf('window.isVaultOpen')).toBe('window')
  })
})
