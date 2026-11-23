/**
 * The plugins of the window slots: the only plugins the window starts with, and the wait the window
 * draws after. Every plugin here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { LoaderHost, PluginParts } from './host'
import { createSignal } from './loading'
import { createTestPluginLoader } from './testing/pluginLoader'

const BAR = 'org.example.ledger'
const TABS = 'org.example.orchard'
const WORKER = 'org.example.harbor'

const MANIFESTS: Record<string, unknown> = {
  [BAR]: {
    id: BAR,
    name: 'Ledger bar',
    version: '1.0.0',
    frontend: {},
    contributes: { slots: [{ slot: 'titleBar', view: 'bar' }] },
  },
  [TABS]: {
    id: TABS,
    name: 'Orchard strip',
    version: '1.0.0',
    frontend: {},
    contributes: { slots: [{ slot: 'statusBar', view: 'strip' }] },
  },
  [WORKER]: { id: WORKER, name: 'Harbor worker', version: '1.0.0', frontend: {} },
}

const createHost = () =>
  ({
    createContext: (pluginId: string) =>
      ({ plugin: { identity: { id: pluginId } }, host: {} }) as unknown as PluginParts,
    disposePlugin: vi.fn<(pluginId: string) => void>(),
    disposeAll: vi.fn(),
  }) satisfies LoaderHost

const createLoader = (
  delays: Record<string, number> = {},
  activations: Record<string, number> = {},
) => {
  const log: string[] = []
  const host = createHost()
  const loader = createTestPluginLoader({
    host,
    fetch: (async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => String(url).includes(`/${one}/`))!
      return new Response(String(url).endsWith('.css') ? '' : JSON.stringify(MANIFESTS[id]))
    }) as unknown as typeof fetch,
    importModule: async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => url.includes(`/${one}/`))!
      log.push(`read ${id}`)
      await new Promise((resolve) => setTimeout(resolve, delays[id] ?? 0))
      return {
        activate: async () => {
          await new Promise((resolve) => setTimeout(resolve, activations[id] ?? 0))
          log.push(`activate ${id}`)
        },
      }
    },
  })
  return { loader, log, host }
}

const startAll = async (loader: ReturnType<typeof createLoader>['loader']) => {
  await loader.trackPlugins([WORKER, TABS, BAR])
  return loader.start()
}

afterEach(() => {
  vi.useRealTimers()
})

describe('the start of the plugins of the window slots', () => {
  it('reads and activates the plugins of the window slots, and leaves the others idle', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader()

    const started = startAll(loader)
    await vi.runAllTimersAsync()
    await started

    expect(log).toContain(`activate ${TABS}`)
    expect(log).toContain(`activate ${BAR}`)
    expect(log).not.toContain(`read ${WORKER}`)
    expect(loader.status.getActivePluginIds().sort()).toStrictEqual([TABS, BAR].sort())
  })

  it('starts a plugin listed while the window runs only where it fills a window slot', async () => {
    vi.useFakeTimers()
    const { loader } = createLoader()
    const started = loader.start()
    await vi.runAllTimersAsync()
    await started

    const tracked = loader.trackPlugins([WORKER, BAR])
    await vi.runAllTimersAsync()
    await tracked

    expect(loader.status.getActivePluginIds()).toStrictEqual([BAR])
  })

  it('settles the wait as soon as the plugins of the window slots started', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader({ [WORKER]: 10_000 })
    let isSettled = false
    const interfaceWait = loader.status.waitForInterface().then(() => (isSettled = true))

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(10)

    expect(isSettled).toBe(true)
    expect(log).toContain(`activate ${BAR}`)
    expect(log).not.toContain(`activate ${WORKER}`)
    await vi.advanceTimersByTimeAsync(10_000)
    await Promise.all([interfaceWait, started])
  })

  it('settles the wait at 3 s, and activates a late plugin once it is read', async () => {
    vi.useFakeTimers()
    const { loader, log } = createLoader({ [BAR]: 3_000 + 5_000 })
    let isSettled = false
    const interfaceWait = loader.status.waitForInterface().then(() => (isSettled = true))

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(3_000 - 1)
    expect(isSettled).toBe(false)
    await vi.advanceTimersByTimeAsync(1)

    expect(isSettled).toBe(true)
    expect(loader.status.isPluginActive(BAR)).toBe(false)
    await vi.advanceTimersByTimeAsync(5_010)
    expect(log).toContain(`activate ${BAR}`)
    expect(loader.status.isPluginActive(BAR)).toBe(true)
    expect(loader.status.getPluginFailure(BAR)).toBeUndefined()
    await Promise.all([interfaceWait, started])
  })

  it('starts the plugins of the window slots after a late one at 3 s without waiting for it', async () => {
    vi.useFakeTimers()
    const { loader } = createLoader({}, { [TABS]: 10_000 })

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(3_000 + 10)

    expect(loader.status.isPluginActive(TABS)).toBe(false)
    expect(loader.status.isPluginActive(BAR)).toBe(true)
    expect(loader.status.isPluginActive(WORKER)).toBe(false)
    await vi.advanceTimersByTimeAsync(10_000)
    expect(loader.status.isPluginActive(TABS)).toBe(true)
    await started
  })
})

describe('a plugin that has not started within 30 s', () => {
  it('counts as failed, and what it registers after is taken back', async () => {
    vi.useFakeTimers()
    const { loader, host } = createLoader({}, { [BAR]: 30_000 + 1_000 })

    const started = startAll(loader)
    await vi.advanceTimersByTimeAsync(30_000 - 100)
    expect(loader.status.getPluginFailure(BAR)).toBeUndefined()
    await vi.advanceTimersByTimeAsync(200)

    expect(loader.status.getPluginFailure(BAR)).toBe('the plugin did not start within 30 s')
    await vi.advanceTimersByTimeAsync(1_000)
    expect(loader.status.isPluginActive(BAR)).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith(BAR)
    expect(loader.status.isPluginActive(TABS)).toBe(true)
    await started
  })
})

it('keeps a module import from a closed window out of the restarted window', async () => {
  const loading = createSignal()
  const finish = createSignal()
  const lateActivate = vi.fn()
  const activate = vi.fn()
  const loader = createTestPluginLoader({
    host: createHost(),
    fetch: async () => new Response(JSON.stringify(MANIFESTS[BAR])),
    importModule: vi
      .fn()
      .mockImplementationOnce(async () => {
        loading.settle()
        await finish.promise
        return { activate: lateActivate }
      })
      .mockResolvedValue({ activate }),
  })
  await loader.trackPlugins([BAR])
  const startup = loader.start()
  await loading.promise
  await loader.close()
  await loader.start()
  finish.settle()
  await startup

  expect(lateActivate).not.toHaveBeenCalled()
  expect(activate).toHaveBeenCalledTimes(1)
  expect(loader.status.isPluginActive(BAR)).toBe(true)
})

it('keeps the last close in effect when it overtakes a restart', async () => {
  const { loader, log } = createLoader()
  await loader.trackPlugins([WORKER])
  await loader.start()
  await loader.close()
  const restart = loader.start()
  const close = loader.close()
  await Promise.all([restart, close])
  await loader.status.startPlugin(WORKER)
  expect(log).toEqual([])
})

it('waits for the new window slots as soon as a restart is requested', async () => {
  const loading = createSignal()
  const finish = createSignal()
  const loader = createTestPluginLoader({
    host: createHost(),
    fetch: async () => new Response(JSON.stringify(MANIFESTS[BAR])),
    importModule: async () => {
      loading.settle()
      await finish.promise
      return {}
    },
  })
  await loader.start()
  await loader.close()
  await loader.trackPlugins([BAR])
  const restart = loader.start()
  let isReady = false
  const wait = loader.status.waitForInterface().then(() => {
    isReady = true
  })
  await loading.promise
  expect(isReady).toBe(false)
  finish.settle()
  await Promise.all([restart, wait])
  expect(isReady).toBe(true)
})
