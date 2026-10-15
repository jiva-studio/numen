import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '../src/testing/mockContext'

describe('mock plugin context', () => {
  it('manages tab kind registrations', () => {
    const ctx = createMockPluginContext({ pluginId: 'tabs-plugin' })
    const d = ctx.registerTabKind({
      kind: 'custom-view',
      component: {},
      title: 'Custom View',
    })

    expect(ctx.getRegisteredTabKinds()).toEqual([
      { kind: 'custom-view', component: {}, title: 'Custom View' },
    ])
    d.dispose()
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('handles command triggers and async handlers', async () => {
    const ctx = createMockPluginContext()
    let executed = false
    ctx.registerCommand({
      id: 'cmd.run',
      title: 'Run',
      handler: async () => {
        executed = true
      },
    })

    await ctx.triggerCommand('cmd.run')
    expect(executed).toBe(true)

    await expect(ctx.triggerCommand('cmd.unknown')).rejects.toThrow(
      'Command not registered: cmd.unknown',
    )
  })

  it('manages agent provider registrations', () => {
    const ctx = createMockPluginContext()
    const d = ctx.registerAgentProvider({
      id: 'custom-llm',
      name: 'Custom LLM',
      createAgent: () => ({}),
      capabilities: ['chat'],
    })

    expect(ctx.getRegisteredAgentProviders()).toHaveLength(1)
    expect(ctx.getRegisteredAgentProviders()[0]?.name).toBe('Custom LLM')
    d.dispose()
    expect(ctx.getRegisteredAgentProviders()).toHaveLength(0)
  })

  it('reacts to setting changes', () => {
    const ctx = createMockPluginContext({
      initialSettings: { timeout: 30 },
    })

    expect(ctx.getSetting('timeout')).toBe(30)

    let notifiedValue: unknown = null
    const d = ctx.onSettingChange('timeout', (val) => {
      notifiedValue = val
    })

    ctx.setSetting('timeout', 60)
    expect(ctx.getSetting('timeout')).toBe(60)
    expect(notifiedValue).toBe(60)

    d.dispose()
    ctx.setSetting('timeout', 90)
    expect(notifiedValue).toBe(60)
  })

  it('cleans up all subscriptions on dispose', () => {
    const ctx = createMockPluginContext()
    ctx.registerCommand({ id: 'c1', title: 'C1', handler: () => {} })
    ctx.registerReader({ id: 'r1', fileExtensions: ['.txt'], component: {} })

    expect(ctx.subscriptions.length).toBe(2)
    ctx.dispose()
    expect(ctx.subscriptions.length).toBe(0)
  })
})
