/**
 * The start of a plugin on use: by a use that waits for it, and by the tabs of its kinds. Every
 * plugin here is invented.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
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


interface LoaderOptions {
  readonly broken?: readonly string[]
  readonly disabled?: readonly string[]
  readonly counts?: Record<string, number>
}

const createLoader = (activations: Record<string, number> = {}, options: LoaderOptions = {}) => {
  const host = {
    createContext: (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
    disposePlugin: vi.fn<(pluginId: string) => void>(),
    disposeAll: vi.fn(),
    getSetting: <T>(pluginId: string, key: string) =>
      (key === 'enabled' && options.disabled?.includes(pluginId) ? false : undefined) as T,
  } satisfies LoaderHost
  return createPluginLoader({
    host,
    fetch: (async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => String(url).includes(`/${one}/`))!
      return new Response(String(url).endsWith('.css') ? '' : JSON.stringify(MANIFESTS[id]))
    }) as unknown as typeof fetch,
    dynamicImport: async (url: string) => {
      const id = Object.keys(MANIFESTS).find((one) => url.includes(`/${one}/`))!
      return {
        activate: async () => {
          if (options.counts) options.counts[id] = (options.counts[id] ?? 0) + 1
          await new Promise((resolve) => setTimeout(resolve, activations[id] ?? 0))
          if (options.broken?.includes(id)) throw new Error('the orchard is closed')
        },
      }
    },
  })
}

const listAll = async (loader: ReturnType<typeof createLoader>) => {
  await loader.trackPlugins([ORCHARD, LEDGER, HARBOR])
  return loader.start()
}

afterEach(() => {
  vi.useRealTimers()
})

describe('a plugin that fills no slot of the window', () => {
  it('stays idle when the window starts', async () => {
    const loader = createLoader()

    await listAll(loader)

    expect(loader.status.getActivePluginIds()).toStrictEqual([])
    expect(loader.status.isPluginStarting(ORCHARD)).toBe(false)
  })

  it('starts on use, and the use waits until it has started', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [ORCHARD]: 1_000 })
    await listAll(loader)
    let isStarted = false

    const use = loader.status.startPlugin(ORCHARD).then(() => (isStarted = true))
    await vi.advanceTimersByTimeAsync(100)
    expect(loader.status.isPluginStarting(ORCHARD)).toBe(true)
    expect(isStarted).toBe(false)
    await vi.advanceTimersByTimeAsync(1_000)
    await use

    expect(isStarted).toBe(true)
    expect(loader.status.isPluginActive(ORCHARD)).toBe(true)
    expect(loader.status.isPluginStarting(ORCHARD)).toBe(false)
  })

  it('starts once for two uses at the same time', async () => {
    const counts: Record<string, number> = {}
    const loader = createLoader({}, { counts })
    await listAll(loader)

    await Promise.all([loader.status.startPlugin(LEDGER), loader.status.startPlugin(LEDGER)])

    expect(counts[LEDGER]).toBe(1)
  })

  it('lets the use go on once the plugin failed, and starts again on the next use', async () => {
    const broken = [ORCHARD]
    const loader = createLoader({}, { broken })
    await listAll(loader)

    await loader.status.startPlugin(ORCHARD)
    expect(loader.status.getPluginFailure(ORCHARD)).toContain('the orchard is closed')
    broken.length = 0
    await loader.status.startPlugin(ORCHARD)

    expect(loader.status.isPluginActive(ORCHARD)).toBe(true)
    expect(loader.status.getPluginFailure(ORCHARD)).toBeUndefined()
  })

  it('lets the use go on once the plugin did not start within the limit', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [LEDGER]: 30_000 * 2 })
    await listAll(loader)
    let isOver = false

    const use = loader.status.startPlugin(LEDGER).then(() => (isOver = true))
    await vi.advanceTimersByTimeAsync(30_000 + 1_000)

    expect(isOver).toBe(true)
    expect(loader.status.isPluginStarting(LEDGER)).toBe(false)
    await vi.runAllTimersAsync()
    await use
  })

  it('does not start on use while it is turned off', async () => {
    const loader = createLoader({}, { disabled: [ORCHARD] })
    await listAll(loader)

    await loader.status.startPlugin(ORCHARD)

    expect(loader.status.isPluginActive(ORCHARD)).toBe(false)
  })

  it('counts as starting until the plugins are listed', () => {
    const loader = createLoader()

    expect(loader.status.isPluginStarting(ORCHARD)).toBe(true)
  })
})

describe('the start of the plugins of tab kinds', () => {
  it('starts only the plugins of the kinds asked for, and waits for them', async () => {
    vi.useFakeTimers()
    const loader = createLoader({ [ORCHARD]: 200, [LEDGER]: 400 })
    await listAll(loader)
    let isSettled = false

    const wait = loader.status.startTabPlugins(['orchard']).then(() => (isSettled = true))
    await vi.advanceTimersByTimeAsync(100)
    expect(isSettled).toBe(false)
    await vi.advanceTimersByTimeAsync(200)
    await wait

    expect(isSettled).toBe(true)
    expect(loader.status.getActivePluginIds()).toStrictEqual([ORCHARD])
  })

  it('starts every plugin with a kind of tab where no kinds are given', async () => {
    const loader = createLoader()
    await listAll(loader)

    await loader.status.startTabPlugins()

    expect(loader.status.getActivePluginIds().sort()).toStrictEqual([LEDGER, ORCHARD].sort())
  })

  it('settles once a plugin of the kinds failed', async () => {
    const loader = createLoader({}, { broken: [ORCHARD] })
    await listAll(loader)

    await loader.status.startTabPlugins(['orchard', 'ledger'])

    expect(loader.status.getPluginFailure(ORCHARD)).toContain('the orchard is closed')
    expect(loader.status.isPluginActive(LEDGER)).toBe(true)
  })

  it('does not start a plugin of the kinds that is turned off', async () => {
    const loader = createLoader({}, { disabled: [LEDGER] })
    await listAll(loader)

    await loader.status.startTabPlugins(['ledger'])

    expect(loader.status.getActivePluginIds()).toStrictEqual([])
  })
})
