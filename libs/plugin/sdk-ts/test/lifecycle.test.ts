import { describe, expect, it, vi } from 'vitest'
import { definePlugin } from '../src/plugin'
import { createMockHostServices, createMockPluginContext } from '../src/testing/mockContext'

describe('plugin lifecycle', () => {
  it('defines and activates a plugin', async () => {
    let activated = false
    const plugin = definePlugin({
      activate(ctx) {
        activated = true
        ctx.register.command({
          id: 'test.hello',
          handler: vi.fn(),
        })
      },
      deactivate() {
        activated = false
      },
    })

    const ctx = createMockPluginContext({ pluginId: 'sample' })
    await plugin.activate(ctx, createMockHostServices())

    expect(activated).toBe(true)
    expect(ctx.getRegisteredCommands()).toHaveLength(1)
    expect(ctx.getRegisteredCommands()[0]?.id).toBe('test.hello')

    await plugin.deactivate?.()
    expect(activated).toBe(false)
  })

  it('activates with host services built apart from the plugin context', async () => {
    const openTab = vi.fn((kind: string, at?: string) => Promise.resolve(`${kind}:${at ?? ''}`))
    const host = createMockHostServices({ openTab })
    const plugin = definePlugin({
      async activate(_plugin, services) {
        await services.tabs.open?.('graph', 'Start.md')
      },
    })

    await plugin.activate(createMockPluginContext(), host)

    expect(openTab).toHaveBeenCalledWith('graph', 'Start.md')
  })

  it('unregisters components when disposed', () => {
    const ctx = createMockPluginContext()
    const disposable = ctx.register.reader({
      id: 'custom-viewer',
      component: {},
    })

    expect(ctx.getRegisteredReaders()).toHaveLength(1)
    disposable.dispose()
    expect(ctx.getRegisteredReaders()).toHaveLength(0)
  })
})
