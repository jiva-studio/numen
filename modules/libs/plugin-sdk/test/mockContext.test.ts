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

  it('keeps a kind that says what its tabs hold, with the hooks it declares', () => {
    const ctx = createMockPluginContext({ pluginId: 'graph-plugin' })
    const navigated: string[] = []
    ctx.registerTabKind<{ here: string }>({
      kind: 'graph',
      component: {},
      open: (at) => ({ here: at }),
      navigate: (state, path) => {
        navigated.push(`${state.here} -> ${path}`)
      },
      isTool: true,
      placement: { pane: 'main', order: 0 },
    })

    const [kind] = ctx.getRegisteredTabKinds()
    const state = kind?.open?.('Start.md', 'graph:1')
    void kind?.navigate?.(state, 'Next.md', 'Next')

    expect(state).toStrictEqual({ here: 'Start.md' })
    expect(navigated).toStrictEqual(['Start.md -> Next.md'])
    expect(kind?.placement).toStrictEqual({ pane: 'main', order: 0 })
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

  it('manages vault change and file select events', () => {
    const ctx = createMockPluginContext()
    let receivedVaultPath = ''
    let receivedSelectedPath = ''

    const vaultSub = ctx.onVaultChange((event) => {
      receivedVaultPath = event.vaultPath ?? ''
    })
    const fileSub = ctx.onFileSelect((event) => {
      receivedSelectedPath = event.path
    })

    ctx.emitVaultChange({ vaultPath: '/path/to/vault' })
    expect(receivedVaultPath).toBe('/path/to/vault')

    ctx.emitFileSelect({ path: 'notes/test.md', title: 'test' })
    expect(receivedSelectedPath).toBe('notes/test.md')

    vaultSub.dispose()
    fileSub.dispose()

    ctx.emitVaultChange({ vaultPath: '/other/vault' })
    ctx.emitFileSelect({ path: 'other/note.md' })
    expect(receivedVaultPath).toBe('/path/to/vault')
    expect(receivedSelectedPath).toBe('notes/test.md')
  })

  it('cleans up all subscriptions on dispose', () => {
    const ctx = createMockPluginContext()
    ctx.registerCommand({ id: 'c1', title: 'C1', handler: () => {} })
    ctx.registerReader({ id: 'r1', fileExtensions: ['.txt'], component: {} })

    expect(ctx.subscriptions.length).toBe(2)
    ctx.dispose()
    expect(ctx.subscriptions.length).toBe(0)
  })

  it('opens tab through openTab method', async () => {
    const ctx = createMockPluginContext()
    const result = await ctx.openTab?.('url', 'https://example.com')
    expect(result).toBe('url:https://example.com')
  })
})
