import { describe, expect, it } from 'vitest'
import { definePlugin } from '../src/plugin'
import { createMockPluginContext } from '../src/testing/mockContext'

describe('plugin lifecycle', () => {
  it('defines and activates a plugin', async () => {
    let activated = false
    const plugin = definePlugin({
      activate(ctx) {
        activated = true
        ctx.registerCommand({
          id: 'test.hello',
          title: 'Hello',
          handler: () => {},
        })
      },
      deactivate() {
        activated = false
      },
    })

    const ctx = createMockPluginContext({ pluginId: 'sample' })
    await plugin.activate(ctx)

    expect(activated).toBe(true)
    expect(ctx.getRegisteredCommands()).toHaveLength(1)
    expect(ctx.getRegisteredCommands()[0]?.id).toBe('test.hello')

    await plugin.deactivate?.()
    expect(activated).toBe(false)
  })

  it('unregisters components when disposed', () => {
    const ctx = createMockPluginContext()
    const disposable = ctx.registerReader({
      id: 'custom-pdf',
      component: {},
      source: 'book',
    })

    expect(ctx.getRegisteredReaders()).toHaveLength(1)
    disposable.dispose()
    expect(ctx.getRegisteredReaders()).toHaveLength(0)
  })
})
