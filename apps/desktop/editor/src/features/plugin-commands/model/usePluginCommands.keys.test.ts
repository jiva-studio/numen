import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import type { PluginHost, PluginPaletteCommand } from '@numen/plugin-loader'
import {
  createCommandRegistry,
  type CommandDeps,
  type CommandTarget,
  type RunSupport,
} from '@/shared/model/command'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createCommandNoteReader, usePluginCommands } from './usePluginCommands'

/** The kinds of files, over a core that answers none. */
const { getFileKinds } = createAnsweredApi({}).vault.file
const RUNS: RunSupport = { canRun: () => true, reportCannotRun: () => undefined }

describe('the palette entry of a command a plugin declares with a Shift shortcut', () => {
  it('is drawn with the keystroke holding Shift that the manifest names, offers welcome ways, and runs registered commands', async () => {
    const shortcuts: Record<string, string> = {
      'md.agent:chat.open': 'mod+shift+a',
      'md.graph:graph.open': 'mod+shift+x',
    }
    const executeCommand = vi.fn(async () => true)
    const paletteList = ref<PluginPaletteCommand[]>([
      { id: 'md.agent:chat.open', title: 'Open agent chat' },
      { id: 'md.graph:graph.open', title: 'Open plex graph', when: 'target.isNote' },
      { id: 'md.items:pick', title: 'Pick item' },
    ])
    const host = {
      commands: {
        hasCommandItems: (id: string) => id === 'md.items:pick',
        getPaletteCommands: () => paletteList.value,
        executeCommand,
      },
      keybindings: { getKeyFor: (id: string) => shortcuts[id] },
    } as unknown as PluginHost

    const registry = createCommandRegistry()
    const pluginCommands = usePluginCommands(
      registry,
      createCommandNoteReader({ readFileKinds: async () => new Map() }),
      host,
    )
    const entries = pluginCommands.getPaletteEntries()

    expect(entries.map((one) => [one.keys?.letter, one.keys?.icons.includes('shift')])).toEqual([
      ['A', true],
      ['X', true],
      [undefined, undefined],
    ])
    expect(entries[2]?.input).toBe('items')
    const noteTarget = {
      path: 'a.md',
      file: 'a.md',
      isReady: true,
      tab: '',
      vault: { id: 'v', name: 'V' },
    } as CommandTarget
    const pdfTarget = {
      path: 'a.pdf',
      file: 'a.pdf',
      isReady: true,
      tab: '',
      vault: { id: 'v', name: 'V' },
    } as CommandTarget
    expect(entries[1]?.isAvailable(noteTarget, RUNS)).toBe(true)
    expect(entries[1]?.isAvailable(pdfTarget, RUNS)).toBe(false)

    expect(pluginCommands.pluginWays.value).toMatchObject([
      { id: 'md.agent:chat.open', text: 'Open agent chat' },
    ])

    const deps = {} as CommandDeps
    const invocationTarget = {
      path: 'a.md',
      file: 'a.md',
      title: 'a',
      isFolder: false,
      others: [],
      tab: 'tab-1',
      vault: { id: 'v', name: 'V' },
      kind: null,
    }
    await registry.get('md.agent:chat.open')?.(
      { id: 'md.agent:chat.open', target: invocationTarget, note: null, name: '' },
      deps,
    )
    await registry.get('md.agent:chat.open')?.(
      {
        id: 'md.agent:chat.open',
        target: invocationTarget,
        note: null,
        name: '',
        values: { prompt: 'hi' },
      },
      deps,
    )
    await registry.get('md.items:pick')?.(
      { id: 'md.items:pick', target: invocationTarget, note: null, name: 'item-1' },
      deps,
    )

    expect(executeCommand).toHaveBeenNthCalledWith(
      1,
      'md.agent:chat.open',
      expect.objectContaining({ path: 'a.md' }),
    )
    expect(executeCommand).toHaveBeenNthCalledWith(
      2,
      'md.agent:chat.open',
      expect.objectContaining({ path: 'a.md' }),
      { values: { prompt: 'hi' } },
    )
    expect(executeCommand).toHaveBeenNthCalledWith(
      3,
      'md.items:pick',
      expect.objectContaining({ path: 'a.md' }),
      { itemId: 'item-1' },
    )

    paletteList.value = []
    await nextTick()
    expect(registry.get('md.agent:chat.open')).toBeUndefined()
  })

  it('returns empty plugin entries and welcome ways when no pluginHost is provided', () => {
    const withoutHost = usePluginCommands(
      createCommandRegistry(),
      createCommandNoteReader({ readFileKinds: getFileKinds }),
      undefined,
    )

    expect(withoutHost.getPaletteEntries()).toStrictEqual([])
    expect(withoutHost.pluginWays.value).toStrictEqual([])
  })
})
