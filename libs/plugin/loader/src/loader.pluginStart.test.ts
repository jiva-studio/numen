/**
 * The start of each plugin as a tab of its kind reads it, and the wait of a new vault for the plugins
 * whose tabs it places. Every plugin here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost } from './host'
import { createPluginLoader } from './loader'

const ORCHARD = 'org.example.orchard'
const LEDGER = 'org.example.ledger'
const HARBOR = 'org.example.harbor'

const MANIFESTS: Record<string, unknown> = {
  [ORCHARD]: {
    id: ORCHARD,
    name: 'Orchard',
    version: '1.0.0',
    frontend: {},
    contributes: { tabKinds: [{ kind: 'orchard', title: 'Orchard' }] },
  },
  [LEDGER]: {
    id: LEDGER,
    name: 'Ledger',
    version: '1.0.0',
    frontend: {},
    contributes: { tabKinds: [{ kind: 'ledger', title: 'Ledger' }] },
  },
  [HARBOR]: { id: HARBOR, name: 'Harbor', version: '1.0.0', frontend: {} },
}

const createLoader = (
  activations: Record<string, number> = {},
  options: { broken?: readonly string[]; disabled?: readonly string[] } = {},
) => {
  const host = {
    createContext: (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
    disposePlugin: vi.fn<(pluginId: string) => void>(),
    disposeAll: vi.fn(),
    getSetting: <T>(pluginId: string, key: string) =>
      (key === 'enabled' && options.disabled?.includes(pluginId) ? false : undefined) as T,
  } satisfies LoaderHost
  const loader = createPluginLoader({
    host,
    fetch: (async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => String(url).includes(`/${one}/`))!
      return new Response(String(url).endsWith('.css') ? '' : JSON.stringify(MANIFESTS[id]))
    }) as unknown as typeof fetch,
    dynamicImport: async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => url.includes(`/${one}/`))!
      return {
        activate: async () => {
          await new Promise((resolve) => setTimeout(resolve, activations[id] ?? 0))
          if (options.broken?.includes(id)) throw new Error('the orchard is closed')
        },
      }
    },
  })
  return loader
}

const startAll = async (loader: ReturnType<typeof createLoader>) => {
  await loader.trackPlugins([ORCHARD, LEDGER, HARBOR])
  return loader.start()
}

afterEach(() => {
  vi.useRealTimers()
})

describe('the start of one plugin', () => {
  it('is going on until the plugins are listed, then until that plugin has started', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [ORCHARD]: 100, [LEDGER]: 5_000 })
    const isOrchardStarting = computed(() => loader.status.isPluginStarting(ORCHARD))
    const isLedgerStarting = computed(() => loader.status.isPluginStarting(LEDGER))

    expect(isOrchardStarting.value).toBe(true)
    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(1_000)

    expect(isOrchardStarting.value).toBe(false)
    expect(isLedgerStarting.value).toBe(true)
    await vi.runAllTimersAsync()
    await started
    expect(isLedgerStarting.value).toBe(false)
  })

  it('is over once the plugin failed', async () => {
    vi.useFakeTimers()
    const loader = createLoader({}, { broken: [ORCHARD] })

    const started = startAll(loader)
    await vi.runAllTimersAsync()
    await started

    expect(loader.status.isPluginStarting(ORCHARD)).toBe(false)
  })

  it('is over once the plugin did not start within the limit', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [LEDGER]: 30_000 * 2 })

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(30_000 + 1_000)

    expect(loader.status.isPluginStarting(LEDGER)).toBe(false)
    await vi.runAllTimersAsync()
    await started
  })

  it('is going on again while a plugin that failed starts once more', async () => {
    vi.useFakeTimers()
    const broken = [ORCHARD]
    const loader = createLoader({ [ORCHARD]: 1_000 }, { broken })
    const started = startAll(loader)
    await vi.runAllTimersAsync()
    await started
    broken.length = 0

    const again = loader.activation.activatePlugin(ORCHARD)
    await vi.advanceTimersByTimeAsync(100)

    expect(loader.status.isPluginStarting(ORCHARD)).toBe(true)
    await vi.runAllTimersAsync()
    expect(await again).toBe(true)
    expect(loader.status.isPluginStarting(ORCHARD)).toBe(false)
  })

  it('is not going on for a plugin that is turned off, once the plugins are listed', async () => {
    vi.useFakeTimers()
    const loader = createLoader({}, { disabled: [ORCHARD] })

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(0)

    expect(loader.status.isPluginStarting(ORCHARD)).toBe(false)
    await vi.runAllTimersAsync()
    await started
  })
})

describe('the wait of a new vault for the plugins of its tabs', () => {
  it('settles once every plugin with a kind of tab has started, while a plugin without one goes on', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [ORCHARD]: 200, [LEDGER]: 400, [HARBOR]: 20_000 })
    let isSettled = false

    const started = startAll(loader)
    const wait = loader.status.waitForTabPlugins().then(() => (isSettled = true))
    await vi.advanceTimersByTimeAsync(300)
    expect(isSettled).toBe(false)
    await vi.advanceTimersByTimeAsync(1_000)

    expect(isSettled).toBe(true)
    expect(loader.status.isPluginStarting(HARBOR)).toBe(true)
    await vi.runAllTimersAsync()
    await Promise.all([started, wait])
  })

  it('settles once a plugin with a kind of tab failed', async () => {
    vi.useFakeTimers()
    const loader = createLoader({}, { broken: [ORCHARD] })
    let isSettled = false

    const started = startAll(loader)
    const wait = loader.status.waitForTabPlugins().then(() => (isSettled = true))
    await vi.runAllTimersAsync()
    await Promise.all([started, wait])

    expect(isSettled).toBe(true)
  })

  it('does not wait for a plugin with a kind of tab that is turned off', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [LEDGER]: 60_000 }, { disabled: [LEDGER] })
    let isSettled = false

    const started = startAll(loader)
    const wait = loader.status.waitForTabPlugins().then(() => (isSettled = true))
    await vi.advanceTimersByTimeAsync(1_000)

    expect(isSettled).toBe(true)
    await vi.runAllTimersAsync()
    await Promise.all([started, wait])
  })
})
