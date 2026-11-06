/**
 * The plugins of the interface: started before the other plugins, and the wait the window draws
 * after. Every plugin here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost } from './host'
import { createPluginLoader } from './loader'
import { INTERFACE_WAIT_MS } from './loading/interfaceStart'

const BAR = 'org.example.ledger'
const TABS = 'org.example.orchard'
const WORKER = 'org.example.harbor'

const MANIFESTS: Record<string, unknown> = {
  [BAR]: {
    id: BAR,
    name: 'Ledger bar',
    version: '1.0.0',
    frontend: {},
    contributes: { views: [{ id: 'bar', slot: 'titleBar' }] },
  },
  [TABS]: {
    id: TABS,
    name: 'Orchard tabs',
    version: '1.0.0',
    frontend: {},
    contributes: { tabKinds: [{ kind: 'orchard', title: 'Orchard' }] },
  },
  [WORKER]: { id: WORKER, name: 'Harbor worker', version: '1.0.0', frontend: {} },
}

const createHost = (): LoaderHost => ({
  createContext: (pluginId) => ({ pluginId }) as unknown as PluginContext,
  disposePlugin: vi.fn(),
  disposeAll: vi.fn(),
})

/** A loader over the invented plugins; `delays` holds the milliseconds each module takes to read. */
const createLoader = (delays: Record<string, number> = {}) => {
  const log: string[] = []
  const loader = createPluginLoader({
    host: createHost(),
    fetch: (async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => String(url).includes(`/${one}/`))!
      return new Response(String(url).endsWith('.css') ? '' : JSON.stringify(MANIFESTS[id]))
    }) as unknown as typeof fetch,
    dynamicImport: async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => url.includes(`/${one}/`))!
      log.push(`read ${id}`)
      await new Promise((resolve) => setTimeout(resolve, delays[id] ?? 0))
      return {
        activate: () => {
          log.push(`activate ${id}`)
        },
      }
    },
  })
  return { loader, log }
}

const startAll = async (loader: ReturnType<typeof createLoader>['loader']) => {
  await loader.trackPlugins([WORKER, TABS, BAR])
  return loader.start()
}

afterEach(() => {
  vi.useRealTimers()
})

describe('the start of the plugins of the interface', () => {
  it('reads and activates the plugins of the interface before the others', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader()

    const started = startAll(loader)
    await vi.runAllTimersAsync()
    await started

    expect(log.indexOf(`activate ${TABS}`)).toBeLessThan(log.indexOf(`read ${WORKER}`))
    expect(log.indexOf(`activate ${BAR}`)).toBeLessThan(log.indexOf(`read ${WORKER}`))
  })

  it('settles the wait as soon as the plugins of the interface started, before the others', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader({ [WORKER]: 10_000 })
    let isSettled = false
    const interfaceWait = loader.waitForInterface().then(() => (isSettled = true))

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(10)

    expect(isSettled).toBe(true)
    expect(log).toContain(`activate ${BAR}`)
    expect(log).not.toContain(`activate ${WORKER}`)
    await vi.advanceTimersByTimeAsync(10_000)
    await Promise.all([interfaceWait, started])
  })

  it('settles the wait at INTERFACE_WAIT_MS without a late plugin, which counts as failed', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader({ [BAR]: INTERFACE_WAIT_MS + 5_000 })
    let isSettled = false
    const interfaceWait = loader.waitForInterface().then(() => (isSettled = true))

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(INTERFACE_WAIT_MS - 1)
    expect(isSettled).toBe(false)
    await vi.advanceTimersByTimeAsync(1)

    expect(isSettled).toBe(true)
    expect(loader.getPluginFailure(BAR)).toMatch(/did not start within 3 s/)
    await vi.advanceTimersByTimeAsync(10_000)
    expect(log).not.toContain(`activate ${BAR}`)
    expect(loader.isPluginActive(BAR)).toBe(false)
    await Promise.all([interfaceWait, started])
  })
})
