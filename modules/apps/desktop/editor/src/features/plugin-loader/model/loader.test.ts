import { describe, expect, it, vi } from 'vitest'
import type { PluginContext, PluginLifecycle } from '@numen/plugin-sdk'
import type { PluginHost } from '@/entities/plugin'
import { createPluginLoader } from './loader'

describe('createPluginLoader', () => {
  function createMockHost(): PluginHost & {
    triggerSettingChange: (pluginId: string, key: string, value: unknown) => void
    settings: Map<string, unknown>
  } {
    const contexts = new Map<string, PluginContext>()
    const settings = new Map<string, unknown>()
    const settingListeners = new Map<string, Set<(val: unknown) => void>>()

    const getSetting: PluginHost['getSetting'] = vi.fn(
      (pluginId: string, key: string) => settings.get(`${pluginId}.${key}`),
    ) as unknown as PluginHost['getSetting']
    const onSettingChange = vi.fn((pluginId: string, key: string, handler: (val: unknown) => void) => {
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
    })

    const triggerSettingChange = async (pluginId: string, key: string, value: unknown) => {
      settings.set(`${pluginId}.${key}`, value)
      const listeners = settingListeners.get(`${pluginId}.${key}`)
      if (listeners) {
        for (const handler of [...listeners]) await handler(value)
      }
    }

    return {
      contextHost: {} as unknown as PluginHost['contextHost'],
      createContext: vi.fn((pluginId: string) => {
        const ctx: PluginContext = {
          pluginId,
          subscriptions: [],
          registerReader: vi.fn(() => ({ dispose: vi.fn() })),
          registerTabKind: vi.fn(() => ({ dispose: vi.fn() })),
          registerCommand: vi.fn(() => ({ dispose: vi.fn() })),
          registerAgentProvider: vi.fn(() => ({ dispose: vi.fn() })),
          getSetting: vi.fn(
            (key: string) => getSetting(pluginId, key),
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

  it('activates and deactivates plugins with lifecycle hooks', async () => {
    const host = createMockHost()
    const loader = createPluginLoader({ host })

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
    const loader = createPluginLoader({ host })

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
    const loader = createPluginLoader({ host })

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
    const loader = createPluginLoader({ host })

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
})
