/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and a run of one with the item picked.
 */
import { describe, expect, it, vi } from 'vitest'
import type { CommandNote, CommandTarget as RunTarget } from '@numen/plugin-loader'
import type { Command } from '@/entities/command'
import type { CommandTarget } from '@/shared/model/command'
import { createCommandItems } from './commandItems'

const FRONT = {
  tab: 'plex:1',
  kind: 'plex',
  path: 'Front.md',
  title: 'Front',
  file: 'Front.md',
  vault: { id: 'v', name: 'V' },
  isReady: true,
} as CommandTarget

const MOVE: Command = {
  id: 'app:tab.moveToWindow',
  text: 'Move to window',
  group: 'window',
  input: 'items',
  isAvailable: () => true,
  listItems: (target) => [{ id: `window-of-${target.tab}`, title: 'Atlas' }],
}
const CLOSE: Command = {
  id: 'app:tab.close',
  text: 'Close',
  group: 'window',
  isAvailable: () => true,
}

const OVER_TAB: RunTarget = { path: '', title: '', isFolder: false, others: [], tab: 'note:2' }

const NOTE: CommandNote = {
  path: '',
  title: '',
  type: 'note',
  isFolder: false,
  others: [],
  tab: 'note:2',
}

const getCommandNote = (target: RunTarget): Promise<CommandNote> =>
  Promise.resolve({ ...NOTE, path: target.path, tab: target.tab })

const createPluginCommands = (
  override: Partial<Parameters<typeof createCommandItems>[0]['pluginCommands']> = {},
) => ({
  hasCommandItems: (id: string) => id === 'org.example.tide:chart.move',
  listCommandItems: vi.fn(async () => [{ id: 'w2', title: 'Second window' }]),
  executeCommand: vi.fn(async () => true),
  ...override,
})

const setup = () => {
  const pluginCommands = createPluginCommands()
  const runApp = vi.fn(async (_id: string, _target: CommandTarget, _itemId: string) => undefined)
  const items = createCommandItems({
    getAppCommands: () => [MOVE, CLOSE],
    pluginCommands,
    getTarget: () => FRONT,
    runApp,
    getCommandNote,
  })
  return { items, pluginCommands, runApp }
}

describe('the items of a command', () => {
  it('are offered by a command of the window with items and by a plugin command with them', () => {
    const { items } = setup()

    expect(items.hasCommandItems(MOVE.id)).toBe(true)
    expect(items.hasCommandItems('org.example.tide:chart.move')).toBe(true)
    expect(items.hasCommandItems(CLOSE.id)).toBe(false)
    expect(items.hasCommandItems('org.example.tide:chart.open')).toBe(false)
  })

  it('of a command of the window are read over the tab the target names', async () => {
    expect(await setup().items.listCommandItems(MOVE.id, OVER_TAB)).toStrictEqual([
      { id: 'window-of-note:2', title: 'Atlas' },
    ])
  })

  it('of a command of the window are read over the file the target names, and no file where it names none', async () => {
    const files: string[] = []
    const items = createCommandItems({
      getAppCommands: () => [
        {
          ...MOVE,
          listItems: async (target) => {
            files.push(target.file)
            return []
          },
        },
      ],
      pluginCommands: createPluginCommands(),
      getTarget: () => FRONT,
      runApp: vi.fn(),
      getCommandNote,
    })

    await items.listCommandItems(MOVE.id, { ...OVER_TAB, path: 'talks/a.mp3' })
    await items.listCommandItems(MOVE.id, OVER_TAB)

    expect(files).toStrictEqual(['talks/a.mp3', ''])
  })

  it('of a command of the window are read over the tab in front where the target names none', async () => {
    expect(await setup().items.listCommandItems(MOVE.id, { ...OVER_TAB, tab: '' })).toStrictEqual([
      { id: 'window-of-plex:1', title: 'Atlas' },
    ])
  })

  it('of a plugin command are the ones the plugin registered', async () => {
    const { items, pluginCommands } = setup()

    expect(await items.listCommandItems('org.example.tide:chart.move', OVER_TAB)).toStrictEqual([
      { id: 'w2', title: 'Second window' },
    ])
    expect(pluginCommands.listCommandItems).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      OVER_TAB,
    )
  })

  it('are none for a command without them', async () => {
    expect(await setup().items.listCommandItems(CLOSE.id, OVER_TAB)).toStrictEqual([])
  })
})

describe('a run with the item picked', () => {
  it('runs a command of the window over the tab of the target, with the item picked', async () => {
    const { items, runApp, pluginCommands } = setup()

    await items.runWithItem(MOVE.id, OVER_TAB, 'w2')

    expect(runApp).toHaveBeenCalledWith(MOVE.id, expect.objectContaining({ tab: 'note:2' }), 'w2')
    expect(pluginCommands.executeCommand).not.toHaveBeenCalled()
  })

  it('runs a plugin command through the plugin, with the item picked', async () => {
    const { items, runApp, pluginCommands } = setup()

    await items.runWithItem('org.example.tide:chart.move', OVER_TAB, 'w2')

    expect(pluginCommands.executeCommand).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      NOTE,
      { itemId: 'w2', isAwaited: true },
    )
    expect(runApp).not.toHaveBeenCalled()
  })
})

describe('a run with the item picked and values', () => {
  it('gives the values to the run of a command of the window and to the run of a plugin command', async () => {
    const { items, runApp, pluginCommands } = setup()

    await items.runWithItem(MOVE.id, OVER_TAB, 'w2', { keep: 'yes' })
    await items.runWithItem('org.example.tide:chart.move', OVER_TAB, 'w2', { keep: 'yes' })

    expect(runApp).toHaveBeenCalledWith(MOVE.id, expect.anything(), 'w2', { keep: 'yes' })
    expect(pluginCommands.executeCommand).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      NOTE,
      { itemId: 'w2', values: { keep: 'yes' }, isAwaited: true },
    )
  })
})

describe('the target the items of a plugin command are read over', () => {
  it('carries the note type and the properties the window reads for its file', async () => {
    const pluginCommands = createPluginCommands({
      hasCommandItems: () => true,
      listCommandItems: vi.fn(async () => []),
    })
    const items = createCommandItems({
      getAppCommands: () => [],
      pluginCommands,
      getTarget: () => FRONT,
      runApp: vi.fn(),
      getCommandNote: async (target) => ({
        ...NOTE,
        path: target.path,
        type: 'link',
        properties: { media: 'a' },
      }),
    })
    const target = { path: 'talk.md', title: 'talk', isFolder: false, others: [], tab: '' }

    await items.listCommandItems('app:file.createText', target)

    expect(pluginCommands.listCommandItems).toHaveBeenCalledWith('app:file.createText', {
      ...target,
      noteType: 'link',
      properties: { media: 'a' },
    })
  })
})
