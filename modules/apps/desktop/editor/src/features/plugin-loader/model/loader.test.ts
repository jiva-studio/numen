import { describe, expect, it, vi } from 'vitest'
import type { PluginContext, PluginLifecycle } from '@numen/plugin-sdk'
import { getPluginStyle, type PluginHost } from '@/entities/plugin'
import { createPluginLoader } from './loader'

describe('createPluginLoader', () => {
  function createMockHost(): PluginHost & {
    triggerSettingChange: (pluginId: string, key: string, value: unknown) => void
    settings: Map<string, unknown>
  } {
    const contexts = new Map<string, PluginContext>()
    const settings = new Map<string, unknown>()
    const settingListeners = new Map<string, Set<(val: unknown) => void>>()

    const getSetting: PluginHost['getSetting'] = vi.fn((pluginId: string, key: string) =>
      settings.get(`${pluginId}.${key}`),
    ) as unknown as PluginHost['getSetting']
    const onSettingChange = vi.fn(
      (pluginId: string, key: string, handler: (val: unknown) => void) => {
        const compositeKey = `${pluginId}.${key}`
        let set = settingListeners.get(compositeKey)
        if (!set) {
          set = new Set()
          settingListeners.set(compositeKey, set)
        }
        set.add(handler)
        return {
          dispose: () => {
            set?.delete(handler)
          },
        }
      },
    )

    const triggerSettingChange = async (pluginId: string, key: string, value: unknown) => {
      settings.set(`${pluginId}.${key}`, value)
      const listeners = settingListeners.get(`${pluginId}.${key}`)
      if (listeners) {
        for (const handler of [...listeners]) await handler(value)
      }
    }

    return {
      contextHost: {} as unknown as PluginHost['contextHost'],
      registerManifest: vi.fn(),
      createContext: vi.fn((pluginId: string, _manifest?: unknown) => {
        const ctx: PluginContext = {
          pluginId,
          subscriptions: [],
          registerReader: vi.fn(() => ({ dispose: vi.fn() })),
          registerTabKind: vi.fn(() => ({ dispose: vi.fn() })),
          registerCommand: vi.fn(() => ({ dispose: vi.fn() })),
          registerAgentProvider: vi.fn(() => ({ dispose: vi.fn() })),
          getSetting: vi.fn((key: string) =>
            getSetting(pluginId, key),
          ) as unknown as PluginContext['getSetting'],
          onSettingChange: vi.fn((key: string, handler: (val: unknown) => void) =>
            onSettingChange(pluginId, key, handler),
          ),
          onVaultChange: vi.fn(() => ({ dispose: vi.fn() })),
          onFileSelect: vi.fn(() => ({ dispose: vi.fn() })),
          emitVaultChange: vi.fn(),
          emitFileSelect: vi.fn(),
        }
        contexts.set(pluginId, ctx)
        return ctx
      }),
      getSetting,
      onSettingChange,
      getPaletteCommands: vi.fn(() => []),
      hasCommand: vi.fn(() => false),
      executeCommand: vi.fn(async () => false),
      handleShortcut: vi.fn(() => false),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
      getRegisteredReaders: vi.fn(() => []),
      getRegisteredTabKinds: vi.fn(() => []),
      getRegisteredCommands: vi.fn(() => []),
      triggerSettingChange,
      settings,
    }
  }

  function createTestLoader(
    host: PluginHost,
    extra: Partial<Omit<Parameters<typeof createPluginLoader>[0], 'host'>> = {},
  ) {
    return createPluginLoader({ host, loaders: {}, ...extra })
  }

  it('activates and deactivates plugins with lifecycle hooks', async () => {
    const host = createMockHost()
    const loader = createTestLoader(host)

    let activated = false
    let deactivated = false

    const mockPluginModule: PluginLifecycle = {
      activate: async (ctx) => {
        expect(ctx.pluginId).toBe('sample.plugin')
        activated = true
      },
      deactivate: async () => {
        deactivated = true
      },
    }

    loader.registerPluginModule('sample.plugin', async () => mockPluginModule)

    expect(loader.isPluginActive('sample.plugin')).toBe(false)

    const success = await loader.activatePlugin('sample.plugin')
    expect(success).toBe(true)
    expect(activated).toBe(true)
    expect(loader.isPluginActive('sample.plugin')).toBe(true)
    expect(loader.getActivePluginIds()).toContain('sample.plugin')

    const deactivatedSuccess = await loader.deactivatePlugin('sample.plugin')
    expect(deactivatedSuccess).toBe(true)
    expect(deactivated).toBe(true)
    expect(loader.isPluginActive('sample.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('sample.plugin')
  })

  it('handles start and close cycles', async () => {
    const host = createMockHost()
    const loader = createTestLoader(host)

    let activatedCount = 0
    loader.registerPluginModule('p1', async () => ({
      activate: () => {
        activatedCount++
      },
    }))

    await loader.start()
    expect(activatedCount).toBe(1)
    expect(loader.isPluginActive('p1')).toBe(true)

    await loader.close()
    expect(loader.isPluginActive('p1')).toBe(false)
    expect(host.disposeAll).toHaveBeenCalled()
  })

  it('does not activate disabled plugins on start', async () => {
    const host = createMockHost()
    host.settings.set('disabled.plugin.enabled', false)
    const loader = createTestLoader(host)

    let activated = false
    loader.registerPluginModule('disabled.plugin', async () => ({
      activate: () => {
        activated = true
      },
    }))

    await loader.start()
    expect(activated).toBe(false)
    expect(loader.isPluginActive('disabled.plugin')).toBe(false)
  })

  it('deactivates and reactivates plugins when enabled setting is toggled', async () => {
    const host = createMockHost()
    const loader = createTestLoader(host)

    let activated = false
    let deactivated = false

    loader.registerPluginModule('toggleable.plugin', async () => ({
      activate: () => {
        activated = true
      },
      deactivate: () => {
        deactivated = true
      },
    }))

    await loader.start()
    expect(activated).toBe(true)
    expect(loader.isPluginActive('toggleable.plugin')).toBe(true)

    // Toggle enabled -> false in settings
    await host.triggerSettingChange('toggleable.plugin', 'enabled', false)

    expect(deactivated).toBe(true)
    expect(loader.isPluginActive('toggleable.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('toggleable.plugin')

    // Toggle enabled -> true in settings
    activated = false
    await host.triggerSettingChange('toggleable.plugin', 'enabled', true)

    expect(activated).toBe(true)
    expect(loader.isPluginActive('toggleable.plugin')).toBe(true)
  })

  it('loads plugin dynamically via manifest URL and mounts/unmounts stylesheet', async () => {
    const host = createMockHost()
    let dynamicImportCalledWith = ''
    let manifestFetchedUrl = ''

    let activated = false
    let deactivated = false

    const mockFetch = vi.fn(async (url: string | URL | Request) => {
      manifestFetchedUrl = String(url)
      return {
        ok: true,
        json: async () => ({
          id: 'dynamic.sample',
          name: 'Dynamic Sample',
          frontend: {
            entrypoint: 'dist/bundle.js',
            styles: 'dist/bundle.css',
          },
        }),
      } as Response
    })

    const mockDynamicImport = vi.fn(async (url: string) => {
      dynamicImportCalledWith = url
      return {
        activate: () => {
          activated = true
        },
        deactivate: () => {
          deactivated = true
        },
      }
    })

    const loader = createTestLoader(host, {
      fetch: mockFetch as unknown as typeof fetch,
      dynamicImport: mockDynamicImport,
    })

    const success = await loader.activatePlugin('dynamic.sample')
    expect(success).toBe(true)
    expect(activated).toBe(true)
    expect(manifestFetchedUrl).toBe('/api/plugins/dynamic.sample/manifest.json')
    expect(dynamicImportCalledWith).toBe('/api/plugins/dynamic.sample/dist/bundle.js')

    expect(document.head.querySelector('link[data-plugin-id="dynamic.sample"]')).toBeNull()
    expect(document.querySelector('link[data-plugin-id="dynamic.sample"]')).toBeNull()
    expect(getPluginStyle('dynamic.sample')).toBe('/api/plugins/dynamic.sample/dist/bundle.css')
    expect(host.createContext).toHaveBeenCalledWith(
      'dynamic.sample',
      expect.objectContaining({ id: 'dynamic.sample' }),
    )

    const deactivateSuccess = await loader.deactivatePlugin('dynamic.sample')
    expect(deactivateSuccess).toBe(true)
    expect(deactivated).toBe(true)

    expect(getPluginStyle('dynamic.sample')).toBeUndefined()
    expect(document.head.querySelector('link[data-plugin-id="dynamic.sample"]')).toBeNull()
  })

  it('handles dynamic load failures gracefully without throwing', async () => {
    const host = createMockHost()
    const mockDynamicImport = vi.fn(async () => {
      throw new Error('Network failure or invalid bundle')
    })

    const loader = createTestLoader(host, {
      dynamicImport: mockDynamicImport,
    })

    const success = await loader.activatePlugin('failing.plugin')
    expect(success).toBe(false)
    expect(loader.isPluginActive('failing.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('failing.plugin')
  })

  it('handles activation throws gracefully and cleans up', async () => {
    const host = createMockHost()
    const mockDynamicImport = vi.fn(async () => ({
      activate: () => {
        throw new Error('Activation error in plugin code')
      },
    }))

    const loader = createTestLoader(host, {
      dynamicImport: mockDynamicImport,
    })

    const success = await loader.activatePlugin('throwing.plugin')
    expect(success).toBe(false)
    expect(loader.isPluginActive('throwing.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('throwing.plugin')
  })
})

describe('createPluginLoader catalog manifests', () => {
  it('hands the catalog manifest to the context of a plugin, on every activation', async () => {
    const createContext = vi.fn((pluginId: string) => ({ pluginId }) as unknown as PluginContext)
    const host = { createContext, disposePlugin: vi.fn(), disposeAll: vi.fn() } as unknown as PluginHost
    const manifest = { id: 'cat.plugin', name: 'Cat', version: '1', permissions: ['vault:read'] }
    const loader = createPluginLoader({
      host,
      loaders: { 'cat.plugin': async () => ({ activate: () => {} }) },
      manifests: { 'cat.plugin': manifest as never },
    })
    await loader.activatePlugin('cat.plugin')
    await loader.deactivatePlugin('cat.plugin')
    await loader.activatePlugin('cat.plugin')
    expect(createContext).toHaveBeenNthCalledWith(1, 'cat.plugin', manifest)
    expect(createContext).toHaveBeenNthCalledWith(2, 'cat.plugin', manifest)
  })
})
