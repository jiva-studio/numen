/**
 * The items a plugin command offers over a target, and the run of the command with the item
 * picked.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import type { CommandNote } from '@numen/plugin-sdk'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

const TIDE = 'org.example.tide'
const TARGET = { path: 'tide/March.md', title: 'March', isFolder: false, others: [], tab: '' }
const NOTE: CommandNote = { ...TARGET, type: 'note' }
const WINDOWS = [
  { id: 'w1', title: 'First window' },
  { id: 'w2', title: 'Second window' },
]

function createHost() {
  const deps: TestHostDeps = {
    getCommandDeclarations: () => [
      { pluginId: TIDE, id: 'chart.move', title: 'Move chart to window' },
      { pluginId: TIDE, id: 'chart.open', title: 'Open chart' },
      { pluginId: TIDE, id: 'chart.share', title: 'Share chart' },
    ],
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
  }
  const host = createTestPluginHost(deps)
  const context = host.lifecycle.createContext(
    TIDE,
    createMockManifest(TIDE, {
      name: 'Tide',
      version: '1',
      commands: ['chart.move', 'chart.open', 'chart.share'],
    }),
  )
  return { host, context }
}

describe('the items of a plugin command', () => {
  it('are the ones its registration gives for the target', async () => {
    const { host, context } = createHost()
    const listItems = vi.fn(() => WINDOWS)
    context.plugin.register.command({ id: 'chart.move', handler: () => undefined, listItems })

    expect(await host.commands.listCommandItems(`${TIDE}:chart.move`, TARGET)).toStrictEqual(
      WINDOWS,
    )
    expect(listItems).toHaveBeenCalledWith(TARGET)
  })

  it('are the ones a registration gives later, as a promise', async () => {
    const { host, context } = createHost()
    context.plugin.register.command({
      id: 'chart.move',
      handler: () => undefined,
      listItems: async () => WINDOWS,
    })

    expect(await host.commands.listCommandItems(`${TIDE}:chart.move`, TARGET)).toStrictEqual(
      WINDOWS,
    )
  })

  it('are none for a command without items, and for one no plugin registered', async () => {
    const { host, context } = createHost()
    context.plugin.register.command({ id: 'chart.open', handler: () => undefined })

    expect(await host.commands.listCommandItems(`${TIDE}:chart.open`, TARGET)).toStrictEqual([])
    expect(await host.commands.listCommandItems(`${TIDE}:chart.share`, TARGET)).toStrictEqual([])
  })

  it('are none when the registration fails to give them', async () => {
    const { host, context } = createHost()
    context.plugin.register.command({
      id: 'chart.move',
      handler: () => undefined,
      listItems: () => {
        throw new Error('the windows could not be listed')
      },
    })

    expect(await host.commands.listCommandItems(`${TIDE}:chart.move`, TARGET)).toStrictEqual([])
  })

  it('reach the handler as its itemId, by the id of the item picked', async () => {
    const { host, context } = createHost()
    const handler = vi.fn()
    context.plugin.register.command({ id: 'chart.move', handler, listItems: () => WINDOWS })

    await host.commands.executeCommand(`${TIDE}:chart.move`, NOTE, { itemId: 'w2' })

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, 'w2', {})
  })

  it('say whether a command offers choices', () => {
    const { host, context } = createHost()
    context.plugin.register.command({
      id: 'chart.move',
      handler: () => undefined,
      listItems: () => [],
    })
    context.plugin.register.command({ id: 'chart.open', handler: () => undefined })

    expect(host.commands.hasCommandItems(`${TIDE}:chart.move`)).toBe(true)
    expect(host.commands.hasCommandItems(`${TIDE}:chart.open`)).toBe(false)
  })

  it('reach the handler with the values of the arguments by name, after the item id', async () => {
    const { host, context } = createHost()
    const handler = vi.fn()
    context.plugin.register.command({ id: 'chart.move', handler, listItems: () => WINDOWS })

    await host.commands.executeCommand(`${TIDE}:chart.move`, NOTE, {
      itemId: 'w2',
      values: { window: 'north' },
    })

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, 'w2', { window: 'north' })
  })

  it('give the values to the handler of a run without an item id, awaited or not', async () => {
    const { host, context } = createHost()
    const handler = vi.fn()
    context.plugin.register.command({ id: 'chart.open', handler })

    await host.commands.executeCommand(`${TIDE}:chart.open`, NOTE, { values: { scale: '2' } })
    await host.commands.executeCommand(`${TIDE}:chart.open`, NOTE, {
      values: { scale: '3' },
      isAwaited: true,
    })

    expect(handler).toHaveBeenNthCalledWith(1, NOTE, undefined, { scale: '2' })
    expect(handler).toHaveBeenNthCalledWith(2, NOTE, undefined, { scale: '3' })
  })

  it('give the handler an empty object when no values were given', async () => {
    const { host, context } = createHost()
    const handler = vi.fn()
    context.plugin.register.command({ id: 'chart.open', handler })

    await host.commands.executeCommand(`${TIDE}:chart.open`, NOTE)

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, undefined, {})
  })
})
