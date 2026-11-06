/**
 * The choices a plugin command offers over a target, and the run of the command with the choice
 * picked.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { CommandNote } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

const TIDE = 'org.example.tide'
const TARGET = { path: 'tide/March.md', title: 'March', isFolder: false, others: [], tab: '' }
const NOTE: CommandNote = { ...TARGET, type: 'note' }
const WINDOWS = [
  { id: 'w1', title: 'First window' },
  { id: 'w2', title: 'Second window' },
]

function createHost() {
  const deps: PluginHostDeps = {
    getCommandDeclarations: () => [
      { pluginId: TIDE, id: 'chart.move', title: 'Move chart to window' },
      { pluginId: TIDE, id: 'chart.open', title: 'Open chart' },
      { pluginId: TIDE, id: 'chart.share', title: 'Share chart' },
    ],
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
  }
  const host = createPluginHost(deps)
  const context = host.createContext(TIDE, createMockManifest(TIDE, { name: 'Tide', version: '1' }))
  return { host, context }
}

describe('the choices of a plugin command', () => {
  it('are the ones its registration gives for the target', async () => {
    const { host, context } = createHost()
    const getChoices = vi.fn(() => WINDOWS)
    context.registerCommand({ id: 'chart.move', handler: () => undefined, getChoices })

    expect(await host.getCommandChoices(`${TIDE}:chart.move`, TARGET)).toStrictEqual(WINDOWS)
    expect(getChoices).toHaveBeenCalledWith(TARGET)
  })

  it('are the ones a registration gives later, as a promise', async () => {
    const { host, context } = createHost()
    context.registerCommand({
      id: 'chart.move',
      handler: () => undefined,
      getChoices: async () => WINDOWS,
    })

    expect(await host.getCommandChoices(`${TIDE}:chart.move`, TARGET)).toStrictEqual(WINDOWS)
  })

  it('are none for a command without choices, and for one no plugin registered', async () => {
    const { host, context } = createHost()
    context.registerCommand({ id: 'chart.open', handler: () => undefined })

    expect(await host.getCommandChoices(`${TIDE}:chart.open`, TARGET)).toStrictEqual([])
    expect(await host.getCommandChoices(`${TIDE}:chart.share`, TARGET)).toStrictEqual([])
  })

  it('are none when the registration fails to give them', async () => {
    const { host, context } = createHost()
    context.registerCommand({
      id: 'chart.move',
      handler: () => undefined,
      getChoices: () => {
        throw new Error('the windows could not be listed')
      },
    })

    expect(await host.getCommandChoices(`${TIDE}:chart.move`, TARGET)).toStrictEqual([])
  })

  it('reach the handler as the id of the choice picked', async () => {
    const { host, context } = createHost()
    const handler = vi.fn()
    context.registerCommand({ id: 'chart.move', handler, getChoices: () => WINDOWS })

    await host.executeCommand(`${TIDE}:chart.move`, NOTE, 'w2')

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, 'w2')
  })

  it('say whether a command offers choices', () => {
    const { host, context } = createHost()
    context.registerCommand({ id: 'chart.move', handler: () => undefined, getChoices: () => [] })
    context.registerCommand({ id: 'chart.open', handler: () => undefined })

    expect(host.hasCommandChoices(`${TIDE}:chart.move`)).toBe(true)
    expect(host.hasCommandChoices(`${TIDE}:chart.open`)).toBe(false)
  })
})
