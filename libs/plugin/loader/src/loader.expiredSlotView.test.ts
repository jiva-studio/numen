/** Review 1687: what a start cancelled at 30_000 registers after the cancel. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createPluginHost, type PluginHostDeps } from './hosting'
import { createPluginLoader } from './loader'

const PLUGIN = 'org.example.ledger'
const MANIFEST = {
  id: PLUGIN,
  name: 'Ledger bar',
  version: '1.0.0',
  contributes: { slots: [{ slot: 'titleBar', placed: { case: 'view', value: 'bar' } }] },
} as unknown as Manifest

function createWindow(activate: (context: PluginContext) => Promise<void>) {
  const deps: PluginHostDeps = {
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [],
    hasPluginFrontend: () => true,
    hasPluginBackend: () => false,
  }
  const host = createPluginHost(deps)
  const loader = createPluginLoader({ host: host.lifecycle, onError: vi.fn() })
  loader.registerPluginManifest(PLUGIN, MANIFEST)
  loader.registerPluginModule(PLUGIN, async () => ({ activate }))
  return { host, loader }
}

afterEach(() => vi.useRealTimers())

describe('a start cancelled at 30 s', () => {
  it('draws no title bar view it registers after the cancel', async () => {
    vi.useFakeTimers()
    const { host, loader } = createWindow(async (context) => {
      await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
      context.register.slotView({ id: 'bar', component: {} })
    })
    const bar = host.slotViews.getSlotView('titleBar')

    const started = loader.start()
    await vi.advanceTimersByTimeAsync(30_000 + 2_000)
    await started

    expect(loader.status.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
    expect(loader.status.isPluginActive(PLUGIN)).toBe(false)
    expect(bar.value).toBeNull()
  })

  it('draws no title bar view once the plugin is turned off after a newer start', async () => {
    vi.useFakeTimers()
    let starts = 0
    const { host, loader } = createWindow(async (context) => {
      starts += 1
      if (starts === 1) {
        await new Promise((resolve) => setTimeout(resolve, 30_000 + 2_000))
      }
      context.register.slotView({ id: 'bar', component: {} })
    })
    const bar = host.slotViews.getSlotView('titleBar')
    const started = loader.start()
    await vi.advanceTimersByTimeAsync(30_000 + 100)
    await started
    expect(await loader.activation.activatePlugin(PLUGIN)).toBe(true)
    await vi.advanceTimersByTimeAsync(2_000)

    await loader.activation.deactivatePlugin(PLUGIN)

    expect(loader.status.isPluginActive(PLUGIN)).toBe(false)
    expect(bar.value).toBeNull()
  })
})

describe('a start cancelled at 30 s that ends while a newer start runs', () => {
  it('reports no command of the newer start as missing', async () => {
    vi.useFakeTimers()
    const reportCommandIssue = vi.fn()
    const host = createPluginHost({
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader: vi.fn(() => vi.fn()),
      openTab: vi.fn(async () => 'tab-1'),
      getCommandDeclarations: () => [
        { pluginId: PLUGIN, id: 'ledger.open', title: 'Open ledger' },
        { pluginId: PLUGIN, id: 'ledger.close', title: 'Close ledger' },
      ],
      hasPluginFrontend: () => true,
      hasPluginBackend: () => false,
      reportCommandIssue,
    })
    const loader = createPluginLoader({ host: host.lifecycle, onError: vi.fn() })
    let starts = 0
    loader.registerPluginModule(PLUGIN, async () => ({
      activate: async (context: PluginContext) => {
        starts += 1
        if (starts === 1) {
          await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
          return
        }
        context.register.command({ id: 'ledger.open', handler: () => undefined })
        await new Promise((resolve) => setTimeout(resolve, 5_000))
        context.register.command({ id: 'ledger.close', handler: () => undefined })
      },
    }))
    const started = loader.start()
    await vi.advanceTimersByTimeAsync(30_000 + 100)
    await started
    const retry = loader.activation.activatePlugin(PLUGIN)

    await vi.advanceTimersByTimeAsync(2_000)
    host.commands.getPaletteCommands()
    await vi.advanceTimersByTimeAsync(5_000)
    await retry

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})
