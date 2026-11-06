/**
 * The items of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with items; and a run of one with the argument picked.
 */
import { describe, expect, it, vi } from 'vitest'
import type { CommandTarget as RunTarget } from '@numen/plugin-host'
import type { Command, CommandTarget } from '@/features/command-palette'
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

const setup = () => {
  const pluginHost = {
    hasCommandItems: (id: string) => id === 'org.example.tide:chart.move',
    listCommandItems: vi.fn(async () => [{ id: 'w2', title: 'Second window' }]),
  }
  const runApp = vi.fn(async (_id: string, _target: CommandTarget, _choice: string) => undefined)
  const runPlugin = vi.fn(async (_id: string, _target: RunTarget, _choice: string) => undefined)
  const items = createCommandItems({
    getAppCommands: () => [MOVE, CLOSE],
    pluginHost,
    getTarget: () => FRONT,
    runApp,
    runPlugin,
  })
  return { items, pluginHost, runApp, runPlugin }
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

  it('of a command of the window are read over the file the target names, or the file in front', async () => {
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
      pluginHost: undefined,
      getTarget: () => FRONT,
      runApp: vi.fn(),
      runPlugin: vi.fn(),
    })

    await items.listCommandItems(MOVE.id, { ...OVER_TAB, path: 'talks/a.mp3' })
    await items.listCommandItems(MOVE.id, OVER_TAB)

    expect(files).toStrictEqual(['talks/a.mp3', 'Front.md'])
  })

  it('of a command of the window are read over the tab in front where the target names none', async () => {
    expect(await setup().items.listCommandItems(MOVE.id, { ...OVER_TAB, tab: '' })).toStrictEqual([
      { id: 'window-of-plex:1', title: 'Atlas' },
    ])
  })

  it('of a plugin command are the ones the plugin registered', async () => {
    const { items, pluginHost } = setup()

    expect(await items.listCommandItems('org.example.tide:chart.move', OVER_TAB)).toStrictEqual([
      { id: 'w2', title: 'Second window' },
    ])
    expect(pluginHost.listCommandItems).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      OVER_TAB,
    )
  })

  it('are none for a command without them', async () => {
    expect(await setup().items.listCommandItems(CLOSE.id, OVER_TAB)).toStrictEqual([])
  })
})

describe('a run with the argument picked', () => {
  it('runs a command of the window over the tab of the target, with the argument', async () => {
    const { items, runApp, runPlugin } = setup()

    await items.runWithArgument(MOVE.id, OVER_TAB, 'w2')

    expect(runApp).toHaveBeenCalledWith(MOVE.id, expect.objectContaining({ tab: 'note:2' }), 'w2')
    expect(runPlugin).not.toHaveBeenCalled()
  })

  it('runs a plugin command through the plugin, with the argument', async () => {
    const { items, runApp, runPlugin } = setup()

    await items.runWithArgument('org.example.tide:chart.move', OVER_TAB, 'w2')

    expect(runPlugin).toHaveBeenCalledWith('org.example.tide:chart.move', OVER_TAB, 'w2')
    expect(runApp).not.toHaveBeenCalled()
  })
})
