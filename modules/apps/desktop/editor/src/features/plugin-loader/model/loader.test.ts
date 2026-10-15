import { describe, expect, it, vi } from 'vitest'
import type { PluginContext, PluginLifecycle } from '@numen/plugin-sdk'
import type { PluginHost } from '@/entities/plugin'
import { createPluginLoader } from './loader'

describe('createPluginLoader', () => {
  function createMockHost(): PluginHost {
    const contexts = new Map<string, PluginContext>()
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
          getSetting: vi.fn(),
          onSettingChange: vi.fn(() => ({ dispose: vi.fn() })),
        }
        contexts.set(pluginId, ctx)
        return ctx
      }),
      getPaletteCommands: vi.fn(() => []),
      hasCommand: vi.fn(() => false),
      executeCommand: vi.fn(async () => false),
      handleShortcut: vi.fn(() => false),
      disposePlugin: vi.fn(),
      disposeAll: vi.fn(),
      getRegisteredReaders: vi.fn(() => []),
      getRegisteredTabKinds: vi.fn(() => []),
      getRegisteredCommands: vi.fn(() => []),
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
})
