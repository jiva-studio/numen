import { describe, expect, it, vi } from 'vitest'
import type { PluginContext, PluginSettings, PluginLifecycle } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createMockNoticeHost, createMockSlotHost } from '@numen/plugin-sdk/testing'
import type { LoaderHost } from './host'
import { getPluginStyle, registerPluginStyle } from './styles'
import { createPluginLoader } from './loader'
import { parseLifecycleHooks, resolveModule } from './module'

function registerModules(
  loader: ReturnType<typeof createPluginLoader>,
  modules: Record<string, () => Promise<object>>,
) {
  for (const [id, load] of Object.entries(modules)) loader.registerPluginModule(id, load)
  return loader
}

describe('createPluginLoader', () => {
  function createMockHost(): LoaderHost & {
    triggerSettingChange: (pluginId: string, key: string, value: unknown) => Promise<void>
    settings: Map<string, unknown>
  } {
    const contexts = new Map<string, PluginContext>()
    const settings = new Map<string, unknown>()
    const settingListeners = new Map<string, Set<(val: unknown) => void | Promise<void>>>()

    const getSetting: NonNullable<LoaderHost['getSetting']> = vi.fn(
      (pluginId: string, key: string) => settings.get(`${pluginId}.${key}`),
    ) as unknown as NonNullable<LoaderHost['getSetting']>
    const onSettingChange = vi.fn(
      (pluginId: string, key: string, handler: (val: unknown) => void | Promise<void>) => {
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
      registerManifest: vi.fn(),
      createContext: vi.fn((pluginId: string, _manifest?: unknown) => {
        const noop = () => ({ dispose: vi.fn() })
        const ctx: PluginContext = {
          plugin: {
            id: pluginId,
            target: 'desktop.test',
            subscriptions: [],
            app: { name: 'numen', version: '0.0.0-test', build: 1, platform: 'linux' },
            openExternal: vi.fn(async () => ({ ok: true as const, value: null })),
          },
          register: {
            slotView: vi.fn(noop),
            reader: vi.fn(noop),
            tabKind: vi.fn(noop),
            command: vi.fn(noop),
            view: vi.fn(noop),
            settingsView: vi.fn(noop),
          },
          settings: {
            get: vi.fn((key: string) =>
              getSetting(pluginId, key),
            ) as unknown as PluginSettings['get'],
            onChange: vi.fn((key: string, handler: (val: unknown) => void) =>
              onSettingChange(pluginId, key, handler),
            ),
          },
          events: {
            onVaultChange: vi.fn(noop),
            onFileSelect: vi.fn(noop),
            emitVaultChange: vi.fn(),
            emitFileSelect: vi.fn(),
          },
          tabs: { open: vi.fn(), navigate: vi.fn(), close: vi.fn() },
          vault: {},
          ui: {
            slots: createMockSlotHost(),
            notices: createMockNoticeHost(),
            mountView: vi.fn(() => ({ focus: vi.fn(), dispose: vi.fn() })),
          },
          work: {},
        }
        contexts.set(pluginId, ctx)
        return ctx
      }),
      getSetting,
      onSettingChange,
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
      triggerSettingChange,
      settings,
    }
  }

  const frontendManifestFetch = (async () =>
    ({
      ok: true,
      text: async () => JSON.stringify({ id: 'x', frontend: {} }),
    }) as Response) as unknown as typeof fetch

  function createTestLoader(
    host: LoaderHost,
    extra: Partial<Omit<Parameters<typeof createPluginLoader>[0], 'host'>> = {},
  ) {
    return createPluginLoader({ host, ...extra })
  }

  it('activates and deactivates plugins with lifecycle hooks', async () => {
    const host = createMockHost()
    const loader = createTestLoader(host)

    let activated = false
    let deactivated = false

    const mockPluginModule: PluginLifecycle = {
      activate: async (ctx) => {
        expect(ctx.plugin.id).toBe('sample.plugin')
        activated = true
      },
      deactivate: async () => {
        deactivated = true
      },
    }

    loader.registerPluginModule('sample.plugin', async () => mockPluginModule)

    expect(loader.status.isPluginActive('sample.plugin')).toBe(false)

    const success = await loader.activation.activatePlugin('sample.plugin')
    expect(success).toBe(true)
    expect(activated).toBe(true)
    expect(loader.status.isPluginActive('sample.plugin')).toBe(true)
    expect(loader.status.getActivePluginIds()).toContain('sample.plugin')

    const deactivatedSuccess = await loader.activation.deactivatePlugin('sample.plugin')
    expect(deactivatedSuccess).toBe(true)
    expect(deactivated).toBe(true)
    expect(loader.status.isPluginActive('sample.plugin')).toBe(false)
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
    expect(loader.status.isPluginActive('p1')).toBe(true)

    await loader.close()
    expect(loader.status.isPluginActive('p1')).toBe(false)
    expect(host.disposeAll).toHaveBeenCalled()

    await host.triggerSettingChange('p1', 'enabled', true)
    expect(loader.status.isPluginActive('p1')).toBe(false)

    await loader.start()
    expect(activatedCount).toBe(2)
    await host.triggerSettingChange('p1', 'enabled', false)
    expect(loader.status.isPluginActive('p1')).toBe(false)
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
    expect(loader.status.isPluginActive('disabled.plugin')).toBe(false)
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
    expect(loader.status.isPluginActive('toggleable.plugin')).toBe(true)

    // Toggle enabled -> false in settings
    await host.triggerSettingChange('toggleable.plugin', 'enabled', false)

    expect(deactivated).toBe(true)
    expect(loader.status.isPluginActive('toggleable.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('toggleable.plugin')

    // Toggle enabled -> true in settings
    activated = false
    await host.triggerSettingChange('toggleable.plugin', 'enabled', true)

    expect(activated).toBe(true)
    expect(loader.status.isPluginActive('toggleable.plugin')).toBe(true)
  })

  it('loads plugin dynamically via manifest URL and mounts/unmounts stylesheet', async () => {
    const host = createMockHost()
    let dynamicImportCalledWith = ''
    let manifestFetchedUrl = ''

    let activated = false
    let deactivated = false

    const mockFetch = vi.fn(async (url: string | URL | Request) => {
      if (String(url).endsWith('manifest.json')) manifestFetchedUrl = String(url)
      return {
        ok: true,
        text: async () =>
          JSON.stringify({
            id: 'dynamic.sample',
            name: 'Dynamic Sample',
            frontend: {},
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

    const success = await loader.activation.activatePlugin('dynamic.sample')
    expect(success).toBe(true)
    expect(activated).toBe(true)
    expect(manifestFetchedUrl).toBe('/plugin/dynamic.sample/manifest.json')
    expect(dynamicImportCalledWith).toBe('/plugin/dynamic.sample/dist/index.js')

    expect(document.head.querySelector('link[data-plugin-id="dynamic.sample"]')).toBeNull()
    expect(document.querySelector('link[data-plugin-id="dynamic.sample"]')).toBeNull()
    expect(getPluginStyle('dynamic.sample')).toBe('/plugin/dynamic.sample/dist/styles.css')
    expect(host.createContext).toHaveBeenCalledWith(
      'dynamic.sample',
      expect.objectContaining({ id: 'dynamic.sample' }),
    )

    const deactivateSuccess = await loader.activation.deactivatePlugin('dynamic.sample')
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

    const success = await loader.activation.activatePlugin('failing.plugin')
    expect(success).toBe(false)
    expect(loader.status.isPluginActive('failing.plugin')).toBe(false)
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

    const success = await loader.activation.activatePlugin('throwing.plugin')
    expect(success).toBe(false)
    expect(loader.status.isPluginActive('throwing.plugin')).toBe(false)
    expect(host.disposePlugin).toHaveBeenCalledWith('throwing.plugin')
  })
})

describe('createPluginLoader catalog manifests', () => {
  it('passes the registered manifest to the context of a plugin', async () => {
    const createContext = vi.fn(
      (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
    )
    const host = {
      createContext,
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
    } as unknown as LoaderHost
    const manifest = createMockManifest('cat.plugin', {
      name: 'Cat',
      version: '1',
      permissions: ['vault:read'],
    })
    const loader = registerModules(createPluginLoader({ host }), {
      'cat.plugin': async () => ({ activate: () => undefined }),
    })
    loader.registerPluginManifest('cat.plugin', manifest)
    await loader.start()
    expect(createContext).toHaveBeenCalledWith('cat.plugin', manifest)
  })
})

describe('createPluginLoader failures, order and toggles', () => {
  function createHost(settings: Record<string, unknown> = {}) {
    const listeners = new Map<string, (value: unknown) => void | Promise<void>>()
    const host = {
      createContext: vi.fn(
        (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
      ),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
      getSetting: vi.fn((id: string, key: string) => settings[`${id}.${key}`]),
      onSettingChange: vi.fn(
        (id: string, _key: string, handler: (value: unknown) => void | Promise<void>) => {
          listeners.set(id, handler)
          return {
            dispose: () => {
              listeners.delete(id)
            },
          }
        },
      ),
    } as unknown as LoaderHost
    return { host, listeners }
  }

  it('shows a plugin whose activate throws as failed with its message and reports it', async () => {
    const { host } = createHost()
    const onError = vi.fn()
    const fault = new Error('activate exploded')
    const loader = registerModules(
      createPluginLoader({
        host,
        onError,
      }),
      {
        bad: async () => ({
          activate: () => {
            throw fault
          },
        }),
      },
    )

    await loader.start()

    expect(loader.status.isPluginActive('bad')).toBe(false)
    expect(loader.status.getPluginFailure('bad')).toBe('activate exploded')
    expect(loader.status.getFailedPlugins()).toStrictEqual([
      { id: 'bad', error: 'activate exploded' },
    ])
    expect(onError).toHaveBeenCalledWith('bad', fault)
  })

  it('clears the failure when the plugin activates later', async () => {
    const { host } = createHost()
    let shouldFail = true
    const loader = registerModules(
      createPluginLoader({
        host,
        onError: vi.fn(),
      }),
      {
        flaky: async () => ({
          activate: () => {
            if (shouldFail) throw new Error('first try')
          },
        }),
      },
    )
    await loader.activation.activatePlugin('flaky')
    shouldFail = false
    await loader.activation.activatePlugin('flaky')

    expect(loader.status.getPluginFailure('flaky')).toBeUndefined()
    expect(loader.status.getFailedPlugins()).toEqual([])
    expect(loader.status.isPluginActive('flaky')).toBe(true)
    expect(await loader.activation.activatePlugin('flaky')).toBe(true)
  })

  it('does not call a failed plugin active when its module cannot be loaded', async () => {
    const { host } = createHost()
    const loader = registerModules(
      createPluginLoader({
        host,
        onError: vi.fn(),
      }),
      {
        gone: async () => {
          throw new Error('chunk missing')
        },
      },
    )
    await loader.activation.activatePlugin('gone')
    expect(loader.status.getPluginFailure('gone')).toBe('chunk missing')
  })

  it('activates in the order the modules were registered, one after the other', async () => {
    const { host } = createHost()
    const log: string[] = []
    const make = (id: string) => async () => ({
      activate: async () => {
        log.push(`${id}:start`)
        await new Promise((resolve) => setTimeout(resolve, 10))
        log.push(`${id}:end`)
      },
    })
    const loader = registerModules(createPluginLoader({ host }), { a: make('a'), b: make('b') })

    await loader.start()

    expect(log).toStrictEqual(['a:start', 'a:end', 'b:start', 'b:end'])
  })

  it('reads every plugin at once, so a slow one does not delay the reading of the others', async () => {
    const { host } = createHost()
    const read: string[] = []
    const make = (id: string) => async () => {
      read.push(id)
      return { activate: () => undefined }
    }
    const loader = registerModules(createPluginLoader({ host }), {
      a: make('a'),
      b: make('b'),
      c: make('c'),
    })

    const started = loader.start()

    await vi.waitFor(() => expect(read).toStrictEqual(['a', 'b', 'c']))
    await started
  })

  it('goes on to the next plugin when one is not read within the wait, and activates it when it is read', async () => {
    vi.useFakeTimers()
    try {
      const { host } = createHost()
      const log: string[] = []
      let finishReading: (module: object) => void = () => undefined
      const slow = () =>
        new Promise<object>((resolve) => {
          finishReading = resolve
        })
      const quick = async () => ({
        activate: () => {
          log.push('quick')
        },
      })
      const loader = registerModules(createPluginLoader({ host }), { slow, quick })

      const started = loader.start()
      await vi.advanceTimersByTimeAsync(3_000)
      expect(log).toStrictEqual(['quick'])
      expect(loader.status.isPluginActive('slow')).toBe(false)

      finishReading({
        activate: () => {
          log.push('slow')
        },
      })
      await started
      expect(log).toStrictEqual(['quick', 'slow'])
      expect(loader.status.isPluginActive('slow')).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('reports a plugin that cannot be read and still starts the others', async () => {
    const { host } = createHost()
    const log: string[] = []
    const loader = registerModules(createPluginLoader({ host }), {
      broken: () => Promise.reject(new Error('chunk missing')),
      fine: async () => ({
        activate: () => {
          log.push('fine')
        },
      }),
    })

    await loader.start()

    expect(loader.status.getPluginFailure('broken')).toBe('chunk missing')
    expect(log).toStrictEqual(['fine'])
  })

  it('activates once when two activations are requested at the same time', async () => {
    const { host } = createHost()
    const activate = vi.fn(async () => {
      await Promise.resolve()
    })
    const loader = registerModules(createPluginLoader({ host }), { p: async () => ({ activate }) })

    await Promise.all([
      loader.activation.activatePlugin('p'),
      loader.activation.activatePlugin('p'),
    ])

    expect(activate).toHaveBeenCalledTimes(1)
  })

  it('settles a wait for a plugin once the plugin has been activated', async () => {
    const { host } = createHost()
    const loader = registerModules(createPluginLoader({ host }), {
      p: async () => ({ activate: () => undefined }),
    })
    const spy = vi.fn()

    const wait = loader.status.settlePlugin('p').then(spy)
    await Promise.resolve()
    expect(spy).not.toHaveBeenCalled()
    await loader.activation.activatePlugin('p')
    await wait

    expect(spy).toHaveBeenCalledOnce()
  })

  it('settles a wait for a plugin at once when the plugin is already active', async () => {
    const { host } = createHost()
    const loader = registerModules(createPluginLoader({ host }), {
      p: async () => ({ activate: () => undefined }),
    })
    await loader.activation.activatePlugin('p')

    await expect(loader.status.settlePlugin('p')).resolves.toBeUndefined()
  })

  it('settles a wait for a plugin whose activation failed', async () => {
    const { host } = createHost()
    const loader = registerModules(createPluginLoader({ host }), {
      p: async () => ({
        activate: () => {
          throw new Error('boom')
        },
      }),
    })
    const wait = loader.status.settlePlugin('p')

    await loader.activation.activatePlugin('p')

    await expect(wait).resolves.toBeUndefined()
  })

  it('settles a wait for a plugin that is never activated after the start limit', async () => {
    vi.useFakeTimers()
    try {
      const { host } = createHost()
      const loader = createPluginLoader({ host })
      const spy = vi.fn()

      const wait = loader.status.settlePlugin('never').then(spy)
      await vi.advanceTimersByTimeAsync(30_000 - 1)
      expect(spy).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(1)
      await wait

      expect(spy).toHaveBeenCalledOnce()
    } finally {
      vi.useRealTimers()
    }
  })

  it('runs a deactivation requested during an activation after it', async () => {
    const { host } = createHost()
    const log: string[] = []
    const loader = registerModules(
      createPluginLoader({
        host,
      }),
      {
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
    )

    await Promise.all([
      loader.activation.activatePlugin('p'),
      loader.activation.deactivatePlugin('p'),
    ])

    expect(log).toStrictEqual(['activate', 'deactivate'])
    expect(loader.status.isPluginActive('p')).toBe(false)
  })

  it('treats a plugin with no enabled setting as enabled', async () => {
    const { host } = createHost()
    const activate = vi.fn()
    const loader = registerModules(createPluginLoader({ host }), { p: async () => ({ activate }) })
    await loader.start()
    expect(activate).toHaveBeenCalledTimes(1)
  })

  it('follows the enabled setting through one subscription per plugin', async () => {
    const { host, listeners } = createHost()
    const loader = registerModules(
      createPluginLoader({
        host,
      }),
      { p: async () => ({ activate: () => undefined }) },
    )
    await loader.start()
    await loader.trackPlugins(['p'])

    expect(host.onSettingChange).toHaveBeenCalledTimes(1)
    await listeners.get('p')?.(false)
    expect(loader.status.isPluginActive('p')).toBe(false)
    await listeners.get('p')?.(true)
    expect(loader.status.isPluginActive('p')).toBe(true)
  })

  /** A loader whose one plugin, `dyn`, is known only by a manifest with a frontend, and the urls it imports. */
  const createDynamicLoader = () => {
    const { host } = createHost()
    const imports: string[] = []
    const loader = createPluginLoader({
      host,
      fetch: (async () =>
        ({
          ok: true,
          text: async () => JSON.stringify({ id: 'dyn', frontend: {} }),
        }) as Response) as unknown as typeof fetch,
      dynamicImport: async (url) => {
        imports.push(url)
        return { activate: () => undefined }
      },
    })
    return { host, imports, loader }
  }

  it('tracks a plugin that is only known by id and loads it from its manifest', async () => {
    const { host, loader } = createDynamicLoader()
    await loader.start()
    await loader.trackPlugins(['dyn'])

    expect(loader.status.isPluginActive('dyn')).toBe(true)
    expect(host.onSettingChange).toHaveBeenCalledTimes(1)
  })

  it('imports a dynamic plugin from its entrypoint url at each activation', async () => {
    const { imports, loader } = createDynamicLoader()
    await loader.activation.activatePlugin('dyn')
    await loader.activation.deactivatePlugin('dyn')
    await loader.activation.activatePlugin('dyn')

    expect(imports).toStrictEqual(['/plugin/dyn/dist/index.js', '/plugin/dyn/dist/index.js'])
  })

  it('neither imports nor fails a plugin whose manifest has no frontend', async () => {
    const { host } = createHost()
    const dynamicImport = vi.fn()
    const loader = createPluginLoader({
      host,
      fetch: (async () =>
        ({
          ok: true,
          text: async () => JSON.stringify({ id: 'ocr' }),
        }) as Response) as unknown as typeof fetch,
      dynamicImport,
    })
    await expect(loader.activation.activatePlugin('ocr')).resolves.toBe(false)
    expect(dynamicImport).not.toHaveBeenCalled()
    expect(loader.status.getPluginFailure('ocr')).toBeUndefined()
  })

  it('fails a dynamic plugin whose manifest.json cannot be fetched', async () => {
    const { host } = createHost()
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      fetch: (async () => ({ ok: false, status: 404 }) as Response) as unknown as typeof fetch,
    })
    await loader.activation.activatePlugin('lost')
    expect(loader.status.getPluginFailure('lost')).toBe(
      'manifest.json of plugin "lost" answered 404',
    )
  })

  it('fails a dynamic plugin whose manifest.json is invalid JSON', async () => {
    const { host } = createHost()
    const loader = createPluginLoader({
      host,
      onError: vi.fn(),
      fetch: (async () =>
        ({
          ok: true,
          text: async () => '{invalid',
        }) as Response) as unknown as typeof fetch,
    })
    await loader.activation.activatePlugin('corrupt')
    expect(loader.status.getPluginFailure('corrupt')).toContain(
      'manifest.json of plugin "corrupt" is invalid:',
    )
  })

  it('cleans up styles on activation failure, non-Error throws, and unactivated deactivations', async () => {
    const { host } = createHost()
    const onError = vi.fn()
    const loader = createPluginLoader({
      host,
      onError,
      fetch: (async () =>
        ({
          ok: true,
          text: async () => JSON.stringify({ id: 'styled.bad', frontend: {} }),
        }) as Response) as unknown as typeof fetch,
      dynamicImport: async () => ({
        activate: () => {
          throw 'string fault'
        },
      }),
    })

    expect(await loader.activation.activatePlugin('styled.bad')).toBe(false)
    expect(getPluginStyle('styled.bad')).toBeUndefined()
    expect(loader.status.getPluginFailure('styled.bad')).toBe(
      'the plugin threw a string, not an Error',
    )

    registerPluginStyle('never.active', '/plugin/never.active/dist/styles.css')
    loader.registerPluginManifest('never.active', createMockManifest('never.active'))
    expect(await loader.activation.deactivatePlugin('never.active')).toBe(false)
    expect(getPluginStyle('never.active')).toBeUndefined()
    expect(host.disposePlugin).toHaveBeenCalledWith('never.active')
    await loader.activation.activatePlugin('never.active', { activate: () => undefined })
    expect(host.createContext).toHaveBeenLastCalledWith('never.active', undefined)
  })

  it('handles deactivation errors and plugins without activate or deactivate hooks', async () => {
    const { host } = createHost()
    const onError = vi.fn()
    const deactivateFault = new Error('deactivate boom')
    let manifestFetches = 0
    const loader = createPluginLoader({
      host,
      onError,
      fetch: (async () => {
        manifestFetches++
        return {
          ok: true,
          text: async () => JSON.stringify({ id: 'dyn.throw', frontend: {} }),
        } as Response
      }) as unknown as typeof fetch,
      dynamicImport: async () => ({
        deactivate: () => {
          throw deactivateFault
        },
      }),
    })

    expect(await loader.activation.activatePlugin('dyn.throw')).toBe(true)
    expect(await loader.activation.deactivatePlugin('dyn.throw')).toBe(true)
    expect(onError).toHaveBeenCalledWith('dyn.throw', deactivateFault)
    expect(loader.status.isPluginActive('dyn.throw')).toBe(false)

    // Re-activating with inline module after deactivation uses undefined manifest because deactivate deleted it
    await loader.activation.activatePlugin('dyn.throw', { activate: () => undefined })
    expect(host.createContext).toHaveBeenLastCalledWith('dyn.throw', undefined)
    await loader.activation.deactivatePlugin('dyn.throw')

    // Re-activating dynamically re-runs loadDynamicModule and passes the newly fetched manifest
    await loader.activation.activatePlugin('dyn.throw')
    expect(manifestFetches).toBe(2)

    // Deactivating a plugin without deactivate hook does not report an error
    loader.registerPluginModule('no.hooks', async () => ({}))
    expect(await loader.activation.activatePlugin('no.hooks')).toBe(true)
    expect(host.createContext).not.toHaveBeenCalledWith('no.hooks', expect.anything())
    expect(await loader.activation.deactivatePlugin('no.hooks')).toBe(true)
    expect(onError).toHaveBeenCalledTimes(1)
  })

  it('handles registerPluginModule and trackPlugins before and after start and close', async () => {
    const { host, listeners } = createHost({
      'disabled.after.enabled': false,
      'disabled.tracked.enabled': false,
    })
    let manifestCount = 0
    const loader = createPluginLoader({
      host,
      fetch: (async (url: string | URL | Request) => {
        manifestCount++
        const isBackendOnly = String(url).includes('backend.only')
        return {
          ok: true,
          text: async () =>
            JSON.stringify(
              isBackendOnly ? { id: 'backend.only' } : { id: 'tracked.one', frontend: {} },
            ),
        } as Response
      }) as unknown as typeof fetch,
      dynamicImport: async () => ({ activate: () => undefined }),
    })

    await loader.trackPlugins(['tracked.one', 'tracked.one', 'backend.only'])
    expect(loader.status.isPluginActive('tracked.one')).toBe(false)
    expect(host.onSettingChange).not.toHaveBeenCalled()

    await loader.start()
    expect(loader.status.isPluginActive('tracked.one')).toBe(true)
    expect(loader.status.isPluginActive('backend.only')).toBe(false)
    expect(manifestCount).toBe(2)

    // Deactivating tracked.one and calling trackPlugins(['tracked.one']) again skips already-tracked id
    await loader.activation.deactivatePlugin('tracked.one')
    await loader.trackPlugins(['tracked.one'])
    expect(loader.status.isPluginActive('tracked.one')).toBe(false)

    // Tracking a disabled plugin after start listens to settings without activating it
    await loader.trackPlugins(['disabled.tracked'])
    expect(loader.status.isPluginActive('disabled.tracked')).toBe(false)
    expect(listeners.has('disabled.tracked')).toBe(true)

    let lateActivated = false
    loader.registerPluginModule('late.enabled', async () => ({
      activate: () => {
        lateActivated = true
      },
    }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(lateActivated).toBe(true)
    expect(listeners.has('late.enabled')).toBe(true)

    let disabledActivated = false
    loader.registerPluginModule('disabled.after', async () => ({
      activate: () => {
        disabledActivated = true
      },
    }))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(disabledActivated).toBe(false)
    expect(listeners.has('disabled.after')).toBe(true)

    await loader.trackPlugins(['disabled.after'])
    await loader.close()

    await loader.trackPlugins(['after.close'])
    expect(loader.status.isPluginActive('after.close')).toBe(false)
  })
})

describe('parseLifecycleHooks and resolveModule', () => {
  it('extracts activate and deactivate from module or default export', () => {
    expect(parseLifecycleHooks(null)).toEqual({})
    expect(parseLifecycleHooks(undefined)).toEqual({})
    expect(parseLifecycleHooks('string')).toEqual({})
    expect(parseLifecycleHooks(42)).toEqual({})

    const activate = () => undefined
    const deactivate = () => undefined
    expect(parseLifecycleHooks({ default: { activate, deactivate } })).toEqual({
      activate,
      deactivate,
    })
    expect(
      parseLifecycleHooks({
        default: { activate: 'not-a-fn', deactivate: 123 },
        activate,
        deactivate,
      }),
    ).toEqual({ activate, deactivate })
    expect(
      parseLifecycleHooks({
        default: null,
        activate: 'not-a-fn',
        deactivate: 'not-a-fn',
      }),
    ).toEqual({ activate: undefined, deactivate: undefined })
  })

  it('resolves candidate modules and loader functions in resolveModule', async () => {
    const ctx = {
      baseUrl: '/plugin',
      fetchFn: vi.fn() as unknown as typeof fetch,
      importFn: vi.fn(),
      manifests: new Map(),
      loaders: new Map(),
    }
    const mod = { activate: () => undefined }
    const defaultMod = { default: mod }
    expect(await resolveModule('p', () => Promise.resolve(mod), ctx)).toBe(mod)
    expect(await resolveModule('p', mod, ctx)).toBe(mod)
    expect(await resolveModule('p', defaultMod, ctx)).toBe(defaultMod)

    const fnWithActivate = Object.assign(() => Promise.resolve('called'), {
      activate: () => undefined,
    })
    expect(await resolveModule('p', fnWithActivate, ctx)).toBe(fnWithActivate)
  })
})

describe('createPluginLoader failure state', () => {
  it('keeps the failure of a plugin that never activated when it is deactivated', async () => {
    const host = {
      createContext: vi.fn(
        (pluginId: string) => ({ plugin: { id: pluginId } }) as unknown as PluginContext,
      ),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
    } as unknown as LoaderHost
    const loader = registerModules(
      createPluginLoader({
        host,
        onError: vi.fn(),
      }),
      {
        bad: async () => ({
          activate: () => {
            throw new Error('no start')
          },
        }),
      },
    )
    await loader.activation.activatePlugin('bad')

    await loader.activation.deactivatePlugin('bad')

    expect(loader.status.getPluginFailure('bad')).toBe('no start')
  })
})
