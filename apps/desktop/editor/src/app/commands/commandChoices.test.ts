/**
 * The choices of a command, whoever owns it: a command of the window that offers them over its
 * target, or a command a plugin registered with choices; and a run of one with the choice picked.
 */
import { describe, expect, it, vi } from 'vitest'
import type { CommandTarget as RunTarget } from '@numen/plugin-host'
import type { Command, CommandTarget } from '@/features/command-palette'
import { createCommandChoices } from './commandChoices'

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
  input: 'choice',
  isAvailable: () => true,
  getChoices: (target) => [{ id: `window-of-${target.tab}`, title: 'Atlas' }],
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
    hasCommandChoices: (id: string) => id === 'org.example.tide:chart.move',
    getCommandChoices: vi.fn(async () => [{ id: 'w2', title: 'Second window' }]),
  }
  const runApp = vi.fn(async (_id: string, _target: CommandTarget, _choice: string) => undefined)
  const runPlugin = vi.fn(async (_id: string, _target: RunTarget, _choice: string) => undefined)
  const choices = createCommandChoices({
    getAppCommands: () => [MOVE, CLOSE],
    pluginHost,
    getTarget: () => FRONT,
    runApp,
    runPlugin,
  })
  return { choices, pluginHost, runApp, runPlugin }
}

describe('the choices of a command', () => {
  it('are offered by a command of the window with choices and by a plugin command with them', () => {
    const { choices } = setup()

    expect(choices.hasCommandChoices(MOVE.id)).toBe(true)
    expect(choices.hasCommandChoices('org.example.tide:chart.move')).toBe(true)
    expect(choices.hasCommandChoices(CLOSE.id)).toBe(false)
    expect(choices.hasCommandChoices('org.example.tide:chart.open')).toBe(false)
  })

  it('of a command of the window are read over the tab the target names', async () => {
    expect(await setup().choices.getCommandChoices(MOVE.id, OVER_TAB)).toStrictEqual([
      { id: 'window-of-note:2', title: 'Atlas' },
    ])
  })

  it('of a command of the window are read over the tab in front where the target names none', async () => {
    expect(
      await setup().choices.getCommandChoices(MOVE.id, { ...OVER_TAB, tab: '' }),
    ).toStrictEqual([{ id: 'window-of-plex:1', title: 'Atlas' }])
  })

  it('of a plugin command are the ones the plugin registered', async () => {
    const { choices, pluginHost } = setup()

    expect(await choices.getCommandChoices('org.example.tide:chart.move', OVER_TAB)).toStrictEqual([
      { id: 'w2', title: 'Second window' },
    ])
    expect(pluginHost.getCommandChoices).toHaveBeenCalledWith(
      'org.example.tide:chart.move',
      OVER_TAB,
    )
  })

  it('are none for a command without them', async () => {
    expect(await setup().choices.getCommandChoices(CLOSE.id, OVER_TAB)).toStrictEqual([])
  })
})

describe('a run with the choice picked', () => {
  it('runs a command of the window over the tab of the target, with the choice', async () => {
    const { choices, runApp, runPlugin } = setup()

    await choices.runChoice(MOVE.id, OVER_TAB, 'w2')

    expect(runApp).toHaveBeenCalledWith(MOVE.id, expect.objectContaining({ tab: 'note:2' }), 'w2')
    expect(runPlugin).not.toHaveBeenCalled()
  })

  it('runs a plugin command through the plugin, with the choice', async () => {
    const { choices, runApp, runPlugin } = setup()

    await choices.runChoice('org.example.tide:chart.move', OVER_TAB, 'w2')

    expect(runPlugin).toHaveBeenCalledWith('org.example.tide:chart.move', OVER_TAB, 'w2')
    expect(runApp).not.toHaveBeenCalled()
  })
})
