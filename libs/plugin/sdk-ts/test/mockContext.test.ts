import { describe, expect, it, vi } from 'vitest'
import { createMockPluginContext } from '../src/testing/mockContext'
import type { PluginVault } from '../src/types'
import type { ConnectionHost } from '../src/connections'
import { CONVERSATION } from '../src/views'

/** Connections that end every call at once, for views that need them to be mounted. */
const CONNECTIONS: ConnectionHost = {
  // eslint-disable-next-line require-yield
  call: async function* () {
    return { ok: true, value: new Uint8Array() }
  },
}

describe('mock plugin context', () => {
  it('manages tab kind registrations', () => {
    const ctx = createMockPluginContext({ pluginId: 'tabs-plugin' })
    const d = ctx.register.tabKind({
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

  it('manages settings view registrations', () => {
    const ctx = createMockPluginContext({ pluginId: 'settings-plugin' })
    const d = ctx.register.settingsView({ id: 'floor', component: {} })

    expect(ctx.getRegisteredSettingsViews()).toEqual([{ id: 'floor', component: {} }])
    d.dispose()
    expect(ctx.getRegisteredSettingsViews()).toHaveLength(0)
  })

  it('removes settings views on dispose', () => {
    const ctx = createMockPluginContext()
    ctx.register.settingsView({ id: 'floor', component: {} })
    ctx.dispose()
    expect(ctx.getRegisteredSettingsViews()).toHaveLength(0)
  })

  it('lists a slot view until it is disposed', () => {
    const ctx = createMockPluginContext({ pluginId: 'bar-plugin' })
    const d = ctx.register.slotView({ id: 'bar', component: {} })

    expect(ctx.getRegisteredSlotViews()).toEqual([{ id: 'bar', component: {} }])
    d.dispose()
    expect(ctx.getRegisteredSlotViews()).toHaveLength(0)
  })

  it('describes the application build', () => {
    const ctx = createMockPluginContext({
      app: { name: 'numen', version: '2026.10.3', build: 42, platform: 'windows' },
    })
    expect(ctx.plugin.app).toEqual({
      name: 'numen',
      version: '2026.10.3',
      build: 42,
      platform: 'windows',
    })
    expect(createMockPluginContext().plugin.app.platform).toBe('linux')
  })

  it('opens web addresses only', async () => {
    const ctx = createMockPluginContext()

    expect(await ctx.plugin.openExternal('https://numen.md/')).toEqual({ ok: true, value: null })
    expect((await ctx.plugin.openExternal('file:///etc/passwd')).ok).toBe(false)
    expect((await ctx.plugin.openExternal('not an address')).ok).toBe(false)
    expect(ctx.getOpenedAddresses()).toEqual(['https://numen.md/'])
  })

  it('keeps a kind that says what its tabs hold, with the hooks it declares', async () => {
    const ctx = createMockPluginContext({ pluginId: 'graph-plugin' })
    const navigated: string[] = []
    ctx.register.tabKind<{ here: string }>({
      kind: 'graph',
      component: {},
      open: (target) => ({ here: target }),
      navigate: (state, path) => {
        navigated.push(`${state.here} -> ${path}`)
      },
      isTool: true,
      placement: { pane: 'main', order: 0 },
    })

    const [kind] = ctx.getRegisteredTabKinds()
    const state = kind?.open?.('Start.md', 'graph:1')
    await kind?.navigate?.(state ?? {}, 'Next.md', 'Next')

    expect(state).toStrictEqual({ here: 'Start.md' })
    expect(navigated).toStrictEqual(['Start.md -> Next.md'])
    expect(kind?.placement).toStrictEqual({ pane: 'main', order: 0 })
  })

  it('handles command triggers and async handlers', async () => {
    const ctx = createMockPluginContext()
    let executed = false
    ctx.register.command({
      id: 'cmd.run',
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

  it('gives a triggered command the item id and the values, empty by default', async () => {
    const ctx = createMockPluginContext()
    const handler = vi.fn()
    ctx.register.command({ id: 'cmd.run', handler })

    await ctx.triggerCommand('cmd.run', undefined, 'w2', { window: 'north' })
    await ctx.triggerCommand('cmd.run')

    expect(handler.mock.calls[0]?.slice(1)).toStrictEqual(['w2', { window: 'north' }])
    expect(handler.mock.calls[1]?.slice(1)).toStrictEqual([undefined, {}])
  })

  it('keeps the views a plugin declares and the views it mounts', () => {
    const ctx = createMockPluginContext()
    const declared = ctx.register.view({ capability: CONVERSATION, component: {} })
    const element = {} as HTMLElement

    const view = ctx.ui.mountView(CONVERSATION, element, {
      connections: CONNECTIONS,
      vault: 'physics',
    })
    view.focus()
    view.dispose()

    expect(ctx.getRegisteredViews()).toStrictEqual([{ capability: CONVERSATION, component: {} }])
    expect(ctx.getMountedViews()).toStrictEqual([
      {
        capability: CONVERSATION,
        element,
        context: { connections: CONNECTIONS, vault: 'physics' },
        isDisposed: true,
        focusCount: 1,
      },
    ])
    declared.dispose()
    expect(ctx.getRegisteredViews()).toHaveLength(0)
  })

  it('reacts to setting changes', () => {
    const ctx = createMockPluginContext({
      initialSettings: { timeout: 30 },
    })

    expect(ctx.settings.get('timeout')).toBe(30)

    let notifiedValue: unknown = null
    const d = ctx.settings.onChange('timeout', (val) => {
      notifiedValue = val
    })

    ctx.setSetting('timeout', 60)
    expect(ctx.settings.get('timeout')).toBe(60)
    expect(notifiedValue).toBe(60)

    d.dispose()
    ctx.setSetting('timeout', 90)
    expect(notifiedValue).toBe(60)
  })

  it('manages vault change and file select events', () => {
    const ctx = createMockPluginContext()
    let receivedVaultPath = ''
    let receivedSelectedPath = ''

    const vaultSub = ctx.events.onVaultChange((event) => {
      receivedVaultPath = event.vaultPath ?? ''
    })
    const fileSub = ctx.events.onFileSelect((event) => {
      receivedSelectedPath = event.path
    })

    ctx.events.emitVaultChange({ vaultPath: '/path/to/vault' })
    expect(receivedVaultPath).toBe('/path/to/vault')

    ctx.events.emitFileSelect({ path: 'notes/test.md', title: 'test' })
    expect(receivedSelectedPath).toBe('notes/test.md')

    vaultSub.dispose()
    fileSub.dispose()

    ctx.events.emitVaultChange({ vaultPath: '/other/vault' })
    ctx.events.emitFileSelect({ path: 'other/note.md' })
    expect(receivedVaultPath).toBe('/path/to/vault')
    expect(receivedSelectedPath).toBe('notes/test.md')
  })

  it('cleans up all subscriptions on dispose', () => {
    const ctx = createMockPluginContext()
    ctx.register.command({ id: 'c1', handler: vi.fn() })
    ctx.register.reader({ id: 'r1', component: {} })

    expect(ctx.plugin.subscriptions.length).toBe(2)
    ctx.dispose()
    expect(ctx.plugin.subscriptions.length).toBe(0)
  })

  it('opens tab through openTab method', async () => {
    const ctx = createMockPluginContext()
    const result = await ctx.tabs.open?.('url', 'https://example.com')
    expect(result).toBe('url:https://example.com')
  })

  it('forwards the provided buffers and closeTab, and closes nothing by default', () => {
    const buffers = { track: () => ({ dispose: vi.fn() }) } as unknown as NonNullable<
      PluginVault['buffers']
    >
    const closeTab = vi.fn()

    const ctx = createMockPluginContext({ buffers, closeTab })

    expect(ctx.vault.buffers).toBe(buffers)
    expect(ctx.tabs.close).toBe(closeTab)
    expect(createMockPluginContext().vault).not.toHaveProperty('buffers')
    expect(() => createMockPluginContext().tabs.close('tab-1')).not.toThrow()
  })
})
