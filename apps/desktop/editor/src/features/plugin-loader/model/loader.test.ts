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
        let activeSet = settingListeners.get(compositeKey)
        if (!activeSet) {
          activeSet = new Set()
          settingListeners.set(compositeKey, activeSet)
        }
        activeSet.add(handler)
        return {
          dispose: () => {
            activeSet?.delete(handler)
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
          registerContextMenu: vi.fn(() => ({ dispose: vi.fn() })),
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
      getRegisteredContextMenus: vi.fn(() => []),
      getContextMenuItems: vi.fn(() => []),
      runContextMenu: vi.fn(async () => true),
      notifyVaultChange: vi.fn(),
      triggerSettingChange,
      settings,
    }
  }

  const frontendManifestFetch = (async () =>
    ({
      ok: true,
      json: async () => ({ id: 'x', frontend: {} }),
    }) as Response) as unknown as typeof fetch

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
    expect(dynamicImportCalledWith).toBe('/api/plugins/dynamic.sample/dist/bundle.js?activation=1')

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
      fetch: frontendManifestFetch,
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
      fetch: frontendManifestFetch,
      dynamicImport: mockDynamicImport,
    })

    const success = await loader.activatePlugin('throwing.plugin')
    expect(success).toBe(false)
    expect(loader.isPluginActive('throwing.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('throwing.plugin')
  })
})

describe('createPluginLoader catalog manifests', () => {
  it('passes the catalog manifest to the context of a plugin, on every activation', async () => {
    const createContext = vi.fn((pluginId: string) => ({ pluginId }) as unknown as PluginContext)
    const host = {
      createContext,
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
    } as unknown as PluginHost
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

describe('createPluginLoader failures, order and toggles', () => {
  function createHost(settings: Record<string, unknown> = {}) {
    const listeners = new Map<string, (value: unknown) => void | Promise<void>>()
    const host = {
      createContext: vi.fn((pluginId: string) => ({ pluginId }) as unknown as PluginContext),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
      getSetting: vi.fn((id: string, key: string) => settings[`${id}.${key}`]),
      onSettingChange: vi.fn(
        (id: string, _key: string, handler: (value: unknown) => void | Promise<void>) => {
          listeners.set(id, handler)
          return { dispose: () => void listeners.delete(id) }
        },
      ),
    } as unknown as PluginHost
    return { host, listeners }
  }

  it('shows a plugin whose activate throws as failed with its message and reports it', async () => {
    const { host } = createHost()
    const onError = vi.fn()
    const fault = new Error('activate exploded')
    const loader = createPluginLoader({
      host,
      onError,
      loaders: {
        bad: async () => ({
          activate: () => {
            throw fault
          },
        }),
      },
    })

    await loader.start()

    expect(loader.isPluginActive('bad')).toBe(false)
    expect(loader.getPluginFailure('bad')).toBe('activate exploded')
    expect(loader.getFailedPlugins()).toStrictEqual([{ id: 'bad', error: 'activate exploded' }])
    expect(onError).toHaveBeenCalledWith('bad', fault)
  })

  it('clears the failure when the plugin activates later', async () => {
    const { host } = createHost()
    let shouldFail = true
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      loaders: {
        flaky: async () => ({
          activate: () => {
            if (shouldFail) throw new Error('first try')
          },
        }),
      },
    })
    await loader.activatePlugin('flaky')
    shouldFail = false
    await loader.activatePlugin('flaky')

    expect(loader.getPluginFailure('flaky')).toBeUndefined()
    expect(loader.isPluginActive('flaky')).toBe(true)
  })

  it('does not call a failed plugin active when its module cannot be loaded', async () => {
    const { host } = createHost()
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      loaders: {
        gone: async () => {
          throw new Error('chunk missing')
        },
      },
    })
    await loader.activatePlugin('gone')
    expect(loader.getPluginFailure('gone')).toBe('chunk missing')
  })

  it('activates in the order of the loaders, one after the other', async () => {
    const { host } = createHost()
    const log: string[] = []
    const make = (id: string) => async () => ({
      activate: async () => {
        log.push(`${id}:start`)
        await Promise.resolve()
        log.push(`${id}:end`)
      },
    })
    const loader = createPluginLoader({ host, loaders: { a: make('a'), b: make('b') } })

    await loader.start()

    expect(log).toStrictEqual(['a:start', 'a:end', 'b:start', 'b:end'])
  })

  it('activates once when two activations are requested at the same time', async () => {
    const { host } = createHost()
    const activate = vi.fn(async () => {
      await Promise.resolve()
    })
    const loader = createPluginLoader({ host, loaders: { p: async () => ({ activate }) } })

    await Promise.all([loader.activatePlugin('p'), loader.activatePlugin('p')])

    expect(activate).toHaveBeenCalledTimes(1)
  })

  it('runs a deactivation requested during an activation after it', async () => {
    const { host } = createHost()
    const log: string[] = []
    const loader = createPluginLoader({
      host,
      loaders: {
        p: async () => ({
          activate: async () => {
            await Promise.resolve()
            log.push('activate')
          },
          deactivate: () => {
            log.push('deactivate')
          },
        }),
      },
    })

    await Promise.all([loader.activatePlugin('p'), loader.deactivatePlugin('p')])

    expect(log).toStrictEqual(['activate', 'deactivate'])
    expect(loader.isPluginActive('p')).toBe(false)
  })

  it('treats a plugin with no enabled setting as enabled', async () => {
    const { host } = createHost()
    const activate = vi.fn()
    const loader = createPluginLoader({ host, loaders: { p: async () => ({ activate }) } })
    await loader.start()
    expect(activate).toHaveBeenCalledTimes(1)
  })

  it('follows the enabled setting through one subscription per plugin', async () => {
    const { host, listeners } = createHost()
    const loader = createPluginLoader({
      host,
      loaders: { p: async () => ({ activate: () => {} }) },
    })
    await loader.start()
    await loader.trackPlugins(['p'])

    expect(host.onSettingChange).toHaveBeenCalledTimes(1)
    await listeners.get('p')?.(false)
    expect(loader.isPluginActive('p')).toBe(false)
    await listeners.get('p')?.(true)
    expect(loader.isPluginActive('p')).toBe(true)
  })

  it('tracks a plugin that is only known by id and loads it from its manifest', async () => {
    const { host } = createHost()
    const imports: string[] = []
    const loader = createPluginLoader({
      host,
      fetch: (async () =>
        ({
          ok: true,
          json: async () => ({ id: 'dyn', frontend: {} }),
        }) as Response) as unknown as typeof fetch,
      dynamicImport: async (url) => {
        imports.push(url)
        return { activate: () => {} }
      },
    })
    await loader.start()
    await loader.trackPlugins(['dyn'])

    expect(loader.isPluginActive('dyn')).toBe(true)
    expect(host.onSettingChange).toHaveBeenCalledTimes(1)
  })

  it('imports a dynamic plugin from a new url at each activation', async () => {
    const { host } = createHost()
    const imports: string[] = []
    const loader = createPluginLoader({
      host,
      fetch: (async () =>
        ({
          ok: true,
          json: async () => ({ id: 'dyn', frontend: {} }),
        }) as Response) as unknown as typeof fetch,
      dynamicImport: async (url) => {
        imports.push(url)
        return { activate: () => {} }
      },
    })
    await loader.activatePlugin('dyn')
    await loader.deactivatePlugin('dyn')
    await loader.activatePlugin('dyn')

    expect(imports).toStrictEqual([
      '/api/plugins/dyn/dist/index.js?activation=1',
      '/api/plugins/dyn/dist/index.js?activation=2',
    ])
  })

  it('neither imports nor fails a plugin whose manifest has no frontend', async () => {
    const { host } = createHost()
    const dynamicImport = vi.fn()
    const loader = createPluginLoader({
      host,
      fetch: (async () =>
        ({ ok: true, json: async () => ({ id: 'ocr' }) }) as Response) as unknown as typeof fetch,
      dynamicImport,
    })
    await expect(loader.activatePlugin('ocr')).resolves.toBe(false)
    expect(dynamicImport).not.toHaveBeenCalled()
    expect(loader.getPluginFailure('ocr')).toBeUndefined()
  })

  it('fails a dynamic plugin whose manifest.json cannot be fetched', async () => {
    const { host } = createHost()
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      fetch: (async () => ({ ok: false, status: 404 }) as Response) as unknown as typeof fetch,
    })
    await loader.activatePlugin('lost')
    expect(loader.getPluginFailure('lost')).toContain('404')
  })
})

describe('createPluginLoader failure state', () => {
  it('keeps the failure of a plugin that never activated when it is deactivated', async () => {
    const host = {
      createContext: vi.fn((pluginId: string) => ({ pluginId }) as unknown as PluginContext),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
    } as unknown as PluginHost
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      loaders: {
        bad: async () => ({
          activate: () => {
            throw new Error('no start')
          },
        }),
      },
    })
    await loader.activatePlugin('bad')

    await loader.deactivatePlugin('bad')

    expect(loader.getPluginFailure('bad')).toBe('no start')
  })
})
