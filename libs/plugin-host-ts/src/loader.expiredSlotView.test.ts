/** Review 1687: what a start cancelled at PLUGIN_START_LIMIT_MS registers after the cancel. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import { createPluginHost } from './hosting/host'
import type { PluginHostDeps } from './hosting/types'
import { createPluginLoader } from './loader'
import { PLUGIN_START_LIMIT_MS } from './loading/limits'

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
  const loader = createPluginLoader({ host, onError: vi.fn() })
  loader.registerPluginManifest(PLUGIN, MANIFEST)
  loader.registerPluginModule(PLUGIN, async () => ({ activate }))
  return { host, loader }
}

afterEach(() => vi.useRealTimers())

describe('a start cancelled at PLUGIN_START_LIMIT_MS', () => {
  it('draws no title bar view it registers after the cancel', async () => {
    vi.useFakeTimers()
    const { host, loader } = createWindow(async (context) => {
      await new Promise((resolve) => setTimeout(resolve, PLUGIN_START_LIMIT_MS + 1_000))
      context.registerSlotView({ id: 'bar', component: {} })
    })
    const bar = host.getSlotView('titleBar')

    const started = loader.start()
    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS + 2_000)
    await started

    expect(loader.getPluginFailure(PLUGIN)).toBe('the plugin did not start within 30 s')
    expect(loader.isPluginActive(PLUGIN)).toBe(false)
    expect(bar.value).toBeNull()
  })

  it('draws no title bar view once the plugin is turned off after a newer start', async () => {
    vi.useFakeTimers()
    let starts = 0
    const { host, loader } = createWindow(async (context) => {
      starts += 1
      if (starts === 1) {
        await new Promise((resolve) => setTimeout(resolve, PLUGIN_START_LIMIT_MS + 2_000))
      }
      context.registerSlotView({ id: 'bar', component: {} })
    })
    const bar = host.getSlotView('titleBar')
    const started = loader.start()
    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS + 100)
    await started
    expect(await loader.activatePlugin(PLUGIN)).toBe(true)
    await vi.advanceTimersByTimeAsync(2_000)

    await loader.deactivatePlugin(PLUGIN)

    expect(loader.isPluginActive(PLUGIN)).toBe(false)
    expect(bar.value).toBeNull()
  })
})

describe('a start cancelled at PLUGIN_START_LIMIT_MS that ends while a newer start runs', () => {
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
    const loader = createPluginLoader({ host, onError: vi.fn() })
    let starts = 0
    loader.registerPluginModule(PLUGIN, async () => ({
      activate: async (context: PluginContext) => {
        starts += 1
        if (starts === 1) {
          await new Promise((resolve) => setTimeout(resolve, PLUGIN_START_LIMIT_MS + 1_000))
          return
        }
        context.registerCommand({ id: 'ledger.open', handler: () => undefined })
        await new Promise((resolve) => setTimeout(resolve, 5_000))
        context.registerCommand({ id: 'ledger.close', handler: () => undefined })
      },
    }))
    const started = loader.start()
    await vi.advanceTimersByTimeAsync(PLUGIN_START_LIMIT_MS + 100)
    await started
    const retry = loader.activatePlugin(PLUGIN)

    await vi.advanceTimersByTimeAsync(2_000)
    host.getPaletteCommands()
    await vi.advanceTimersByTimeAsync(5_000)
    await retry

    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})
