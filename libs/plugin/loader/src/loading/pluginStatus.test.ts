import { describe, expect, it, vi } from 'vitest'
import type { PluginContext } from '@numen/plugin-sdk'
import type { LoaderHost } from '../host'
import { createPluginLoader } from '../loader'
import { createPluginStatus } from './pluginStatus'

const ORCHARD = 'org.example.orchard'

const createHost = (): LoaderHost => ({
  createContext: (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
  disposePlugin: vi.fn(),
  disposeAll: vi.fn(),
})

const createLoader = (status: ReturnType<typeof createPluginStatus>, isBroken = false) =>
  createPluginLoader({
    host: createHost(),
    status,
    fetch: (async (url: string) =>
      new Response(
        String(url).endsWith('.css')
          ? ''
          : JSON.stringify({ id: ORCHARD, name: 'Orchard', version: '1.0.0', frontend: {} }),
      )) as unknown as typeof fetch,
    dynamicImport: async () => ({
      activate: async () => {
        if (isBroken) throw new Error('the orchard is closed')
      },
    }),
  })

describe('the status made before the loader', () => {
  it('counts every plugin as starting until the loader runs', () => {
    const status = createPluginStatus({ isEnabled: () => true })

    expect(status.isPluginStarting(ORCHARD)).toBe(true)
    expect(status.getPluginFailure(ORCHARD)).toBeUndefined()
  })

  it('is over for a plugin the loader started on use', async () => {
    const status = createPluginStatus({ isEnabled: () => true })
    const loader = createLoader(status)

    await loader.trackPlugins([ORCHARD])
    await loader.start()
    await status.startPlugin(ORCHARD)

    expect(status.isPluginStarting(ORCHARD)).toBe(false)
    expect(status.state.activePlugins.value.has(ORCHARD)).toBe(true)
    expect(loader.status.getActivePluginIds()).toStrictEqual([ORCHARD])
  })

  it('holds the failure of a plugin that did not start', async () => {
    const status = createPluginStatus({ isEnabled: () => true })
    const loader = createLoader(status, true)

    await loader.trackPlugins([ORCHARD])
    await loader.start()
    await status.startPlugin(ORCHARD)

    expect(status.getPluginFailure(ORCHARD)).toContain('the orchard is closed')
    expect(loader.status.getPluginFailure(ORCHARD)).toBe(status.getPluginFailure(ORCHARD))
  })

  it('does not count a disabled plugin as starting once the loader runs', async () => {
    const status = createPluginStatus({ isEnabled: () => false })
    const loader = createLoader(status)

    await loader.trackPlugins([ORCHARD])
    await loader.start()

    expect(status.isPluginStarting(ORCHARD)).toBe(false)
    expect(status.state.activePlugins.value.size).toBe(0)
  })
})
