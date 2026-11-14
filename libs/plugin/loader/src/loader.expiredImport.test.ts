/** A start that passes its limit while the module of the plugin is still being read. The plugin is invented. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost, PluginParts } from './host'
import { createPluginLoader } from './loader'

const HARBOR = 'org.example.harbor'
const MANIFEST = { id: HARBOR, name: 'Harbor', version: '1.0.0', frontend: {} }

afterEach(() => {
  vi.useRealTimers()
})

describe('a plugin whose module is read after the start limit', () => {
  it('counts as failed and never becomes active', async () => {
    vi.useFakeTimers()
    const activate = vi.fn()
    const host = {
      createContext: (pluginId: string) =>
        ({ plugin: { identity: { id: pluginId } }, host: {} }) as unknown as PluginParts,
      disposePlugin: vi.fn<(pluginId: string) => void>(),
      disposeAll: vi.fn(),
      getSetting: <T>() => undefined as T,
    } satisfies LoaderHost
    const loader = createPluginLoader({
      host,
      fetch: (async (url: string) =>
        new Response(
          String(url).endsWith('.css') ? '' : JSON.stringify(MANIFEST),
        )) as unknown as typeof fetch,
      dynamicImport: async () => {
        await new Promise((resolve) => setTimeout(resolve, 30_000 + 1_000))
        return { activate }
      },
    })

    await loader.trackPlugins([HARBOR])
    await loader.start()
    const started = loader.status.startPlugin(HARBOR)
    await vi.advanceTimersByTimeAsync(30_000 + 2_000)
    await vi.runAllTimersAsync()
    await started

    expect(loader.status.getPluginFailure(HARBOR)).toBe('the plugin did not start within 30 s')
    expect(activate).not.toHaveBeenCalled()
  })
})

describe('a plugin that started within the limit', () => {
  it('registers what it registers after the limit', async () => {
    vi.useFakeTimers()
    const registerCommand = vi.fn(() => ({ dispose: vi.fn() }))
    let context: PluginContext | undefined
    const host = {
      createContext: (pluginId: string) =>
        ({
          plugin: { identity: { id: pluginId }, register: { command: registerCommand } },
          host: {},
        }) as unknown as PluginParts,
      disposePlugin: vi.fn<(pluginId: string) => void>(),
      disposeAll: vi.fn(),
      getSetting: <T>() => undefined as T,
    } satisfies LoaderHost
    const loader = createPluginLoader({
      host,
      fetch: (async (url: string) =>
        new Response(
          String(url).endsWith('.css') ? '' : JSON.stringify(MANIFEST),
        )) as unknown as typeof fetch,
      dynamicImport: async () => ({
        activate: (one: PluginContext) => {
          context = one
        },
      }),
    })

    await loader.trackPlugins([HARBOR])
    await loader.start()
    await loader.status.startPlugin(HARBOR)
    await vi.advanceTimersByTimeAsync(30_000 + 1_000)
    context?.register.command({ id: 'harbor.open', handler: () => undefined })

    expect(loader.status.isPluginActive(HARBOR)).toBe(true)
    expect(registerCommand).toHaveBeenCalledTimes(1)
  })
})
