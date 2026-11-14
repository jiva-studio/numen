import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginParts } from './context'
import type { PluginContextHost } from './types'
import { createTestContextHost } from '../testing/contextHost'
import { STUB_COMPONENT } from '../testing/stubComponent'

describe('PluginContext', () => {
  function createMockHost(): { host: PluginContextHost; calls: Record<string, number> } {
    const calls = {
      registerReader: 0,
      registerTabKind: 0,
      registerCommand: 0,
      getSetting: 0,
      onSettingChange: 0,
      onVaultChange: 0,
      onFileSelect: 0,
      emitVaultChange: 0,
      emitFileSelect: 0,
    }

    const host: PluginContextHost = createTestContextHost({
      info: { openExternal: vi.fn(async (_url: string) => undefined) },
      registration: {
        registerReader: vi.fn((_id, _reg) => {
          calls.registerReader++
          return { dispose: vi.fn() }
        }),
        registerSettingsView: vi.fn((_id, _reg) => ({ dispose: vi.fn() })),
        registerTabKind: vi.fn((_id, _reg) => {
          calls.registerTabKind++
          return { dispose: vi.fn() }
        }),
        registerCommand: vi.fn((_id, _reg) => {
          calls.registerCommand++
          return { dispose: vi.fn() }
        }),
      },
      views: {
        registerSlotView: vi.fn((_id, _slot, _reg) => ({ dispose: vi.fn() })),
        registerView: vi.fn((_id, _reg) => ({ dispose: vi.fn() })),
        mountView: vi.fn((_capability, _element, _context) => ({
          focus: vi.fn(),
          dispose: vi.fn(),
        })),
        registerDropHandler: vi.fn((_id, _reg) => ({ dispose: vi.fn() })),
      },
      settings: {
        getSetting: vi.fn((_id: string, key: string) => {
          calls.getSetting++
          return key === 'enabled' ? true : undefined
        }) as PluginContextHost['settings']['getSetting'],
        onSettingChange: vi.fn((_id, _key, _handler) => {
          calls.onSettingChange++
          return { dispose: vi.fn() }
        }),
      },
      events: {
        onVaultChange: vi.fn((_id, _handler) => {
          calls.onVaultChange++
          return { dispose: vi.fn() }
        }),
        onFileSelect: vi.fn((_id, _handler) => {
          calls.onFileSelect++
          return { dispose: vi.fn() }
        }),
        emitVaultChange: vi.fn((_id, _event) => {
          calls.emitVaultChange++
        }),
        emitFileSelect: vi.fn((_id, _event) => {
          calls.emitFileSelect++
        }),
      },
      tabs: { openTab: vi.fn(async (_kind, _at) => 'opened-tab-id') },
    })

    return { host, calls }
  }

  it('tracks subscriptions and delegates registrations to host', async () => {
    const { host } = createMockHost()
    const ctx = createPluginParts(
      'test.plugin',
      host,
      createMockManifest('test.plugin', {
        name: 'Test Plugin',
        permissions: ['vault:read', 'vault:write'],
        readers: ['pdf'],
        commands: ['cmd1'],
        tabKinds: ['custom-tab'],
        capabilities: ['tools'],
      }),
    )

    expect(ctx.plugin.identity.id).toBe('test.plugin')
    expect(ctx.plugin.identity.subscriptions).toHaveLength(0)

    const sub1 = ctx.plugin.register.reader({
      id: 'pdf',
      component: STUB_COMPONENT,
    })
    expect(ctx.plugin.identity.subscriptions).toHaveLength(1)
    expect(host.registration.registerReader).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ id: 'pdf' }),
    )

    const sub2 = ctx.plugin.register.command({
      id: 'cmd1',
      handler: vi.fn(() => Promise.resolve()),
    })
    expect(ctx.plugin.identity.subscriptions).toHaveLength(2)
    expect(host.registration.registerCommand).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ id: 'cmd1' }),
    )

    const sub3 = ctx.plugin.register.tabKind({
      kind: 'custom-tab',
      component: STUB_COMPONENT,
    })
    expect(ctx.plugin.identity.subscriptions).toHaveLength(3)
    expect(host.registration.registerTabKind).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ kind: 'custom-tab' }),
    )

    expect(ctx.plugin.settings.get('enabled')).toBe(true)
    expect(host.settings.getSetting).toHaveBeenCalledWith('test.plugin', 'enabled')

    const sub4 = ctx.plugin.settings.onChange('enabled', vi.fn())
    expect(ctx.plugin.identity.subscriptions).toHaveLength(4)
    expect(host.settings.onSettingChange).toHaveBeenCalledWith(
      'test.plugin',
      'enabled',
      expect.any(Function),
    )

    const sub5 = ctx.host.events.onVaultChange(vi.fn())
    expect(ctx.plugin.identity.subscriptions).toHaveLength(5)

    const sub6 = ctx.host.events.onFileSelect(vi.fn())
    expect(ctx.plugin.identity.subscriptions).toHaveLength(6)

    ctx.host.events.emitVaultChange({ vaultPath: '/path' })
    expect(host.events.emitVaultChange).toHaveBeenCalledWith('test.plugin', { vaultPath: '/path' })

    ctx.host.events.emitFileSelect({ path: '/file.md' })
    expect(host.events.emitFileSelect).toHaveBeenCalledWith('test.plugin', { path: '/file.md' })

    const openedTab = await ctx.host.tabs.open?.('url', 'https://example.com')
    expect(openedTab).toBe('opened-tab-id')
    expect(host.tabs.openTab).toHaveBeenCalledWith('url', 'https://example.com')

    expect(ctx.host.tabs.navigate?.('url', 'https://example.com')).toBeUndefined()
    expect(() => ctx.host.tabs.close?.('tab-1')).not.toThrow()

    const navigateTab = vi.fn(async () => 'tab-new')
    const closeTab = vi.fn()
    const hostWithTabs = createPluginParts(
      'tabs.plugin',
      createTestContextHost({ tabs: { navigateTab, closeTab } }),
      { permissions: ['vault:read', 'vault:write'] },
    )
    await expect(hostWithTabs.host.tabs.navigate?.('note', 'a.md')).resolves.toBe('tab-new')
    expect(navigateTab).toHaveBeenCalledWith('note', 'a.md')
    hostWithTabs.host.tabs.close?.('tab-9')
    expect(closeTab).toHaveBeenCalledWith('tab-9')

    const ctxWithoutOpen = createPluginParts('noopen.plugin', createTestContextHost(), {
      permissions: ['vault:read'],
    })
    expect(ctxWithoutOpen.host.tabs.open?.('note', 'a.md')).toBeUndefined()

    const mountedEl = document.createElement('div')
    const mounted = ctx.host.ui.mountView('note-editor', mountedEl, { path: 'a.md' })
    mounted.focus()
    const mountResult = vi.mocked(host.views.mountView).mock.results[0]?.value
    expect(mountResult.focus).toHaveBeenCalledTimes(1)
    mounted.dispose()

    expect(ctx.plugin.identity.hasPermission?.('vault:read')).toBe(true)
    expect(ctx.plugin.identity.hasPermission?.('unknown:perm')).toBe(false)

    // Calling dispose twice on sub1 must not splice another subscription
    sub1.dispose()
    expect(ctx.plugin.identity.subscriptions).toHaveLength(5)
    sub1.dispose()
    expect(ctx.plugin.identity.subscriptions).toHaveLength(5)
    sub2.dispose()
    sub3.dispose()
    sub4.dispose()
    sub5.dispose()
    sub6.dispose()
    expect(ctx.plugin.identity.subscriptions).toHaveLength(0)
  })

  it('enforces vault:read permission for onVaultChange and onFileSelect, and vault:write for emitFileSelect', () => {
    const { host } = createMockHost()
    const ctxNoPerms = createPluginParts('restricted.plugin', host, {
      id: 'restricted.plugin',
      name: 'Restricted',
      version: '1.0.0',
      permissions: [],
    })

    expect(() => ctxNoPerms.host.events.onVaultChange(vi.fn())).toThrowError(
      'Plugin "restricted.plugin" lacks required permission "vault:read" for onVaultChange',
    )
    expect(() => ctxNoPerms.host.events.onFileSelect(vi.fn())).toThrowError(
      'Plugin "restricted.plugin" lacks required permission "vault:read" for onFileSelect',
    )
    expect(() => ctxNoPerms.host.events.emitFileSelect({ path: '/test.md' })).toThrowError(
      'Plugin "restricted.plugin" lacks required permission "vault:write" for emitFileSelect',
    )

    const ctxUndefinedManifest = createPluginParts('no.manifest', host)
    expect(ctxUndefinedManifest.plugin.identity.hasPermission?.('vault:read')).toBe(false)
    expect(() => ctxUndefinedManifest.host.events.onVaultChange(vi.fn())).toThrowError(
      'Plugin "no.manifest" lacks required permission "vault:read" for onVaultChange',
    )

    const ctxEmptyManifest = createPluginParts('empty.manifest', host, {})
    expect(ctxEmptyManifest.plugin.identity.hasPermission?.('vault:read')).toBe(false)
    expect(() => ctxEmptyManifest.host.events.onVaultChange(vi.fn())).toThrowError(
      'Plugin "empty.manifest" lacks required permission "vault:read" for onVaultChange',
    )

    const ctxWithRead = createPluginParts('reader.plugin', host, {
      id: 'reader.plugin',
      name: 'Reader',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(() => ctxWithRead.host.events.onVaultChange(vi.fn())).not.toThrow()
    expect(() => ctxWithRead.host.events.onFileSelect(vi.fn())).not.toThrow()
    expect(() => ctxWithRead.host.events.emitFileSelect({ path: '/test.md' })).toThrowError(
      'Plugin "reader.plugin" lacks required permission "vault:write" for emitFileSelect',
    )
  })

  it('enforces vault:write permission for emitVaultChange', () => {
    const { host } = createMockHost()
    const ctxReadOnly = createPluginParts('readonly.plugin', host, {
      id: 'readonly.plugin',
      name: 'Read Only',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(() => ctxReadOnly.host.events.emitVaultChange({ vaultPath: '/test' })).toThrowError(
      'Plugin "readonly.plugin" lacks required permission "vault:write" for emitVaultChange',
    )

    const ctxWrite = createPluginParts('writer.plugin', host, {
      id: 'writer.plugin',
      name: 'Writer',
      version: '1.0.0',
      permissions: ['vault:write'],
    })

    expect(() => ctxWrite.host.events.emitVaultChange({ vaultPath: '/test' })).not.toThrow()
    expect(host.events.emitVaultChange).toHaveBeenCalledWith('writer.plugin', {
      vaultPath: '/test',
    })
  })

  it('registers a view in the slot its manifest places the id in', () => {
    const { host } = createMockHost()
    const ctx = createPluginParts('org.example.orchard', host, {
      id: 'org.example.orchard',
      name: 'Orchard',
      version: '1.0.0',
      contributes: { slots: [{ slot: 'titleBar', placed: { case: 'view', value: 'bar' } }] },
    } as unknown as Parameters<typeof createPluginParts>[2])

    ctx.plugin.register.slotView({ id: 'bar', component: STUB_COMPONENT })

    expect(ctx.plugin.identity.subscriptions).toHaveLength(1)
    expect(host.views.registerSlotView).toHaveBeenCalledWith(
      'org.example.orchard',
      'titleBar',
      expect.objectContaining({ id: 'bar' }),
    )
  })

  it('rejects a view its manifest places nowhere', () => {
    const { host } = createMockHost()
    const ctx = createPluginParts('org.example.orchard', host)

    expect(() => ctx.plugin.register.slotView({ id: 'bar', component: STUB_COMPONENT })).toThrow(
      /places nowhere/,
    )
    expect(host.views.registerSlotView).not.toHaveBeenCalled()
  })

  it('reports the application build the host runs', () => {
    const { host } = createMockHost()

    expect(createPluginParts('test.plugin', host).plugin.identity.app).toEqual({
      name: 'numen',
      version: '2026.10.1',
      build: 884,
      platform: 'linux',
    })
  })

  it('opens a web address through the host and refuses any other', async () => {
    const { host } = createMockHost()
    const ctx = createPluginParts(
      'test.plugin',
      host,
      createMockManifest('test.plugin', { permissions: ['link:open'] }),
    )

    expect(await ctx.plugin.identity.openExternal('https://numen.md/')).toEqual({
      ok: true,
      value: null,
    })
    expect((await ctx.plugin.identity.openExternal('file:///etc/passwd')).ok).toBe(false)
    expect(host.info.openExternal).toHaveBeenCalledTimes(1)
    expect(host.info.openExternal).toHaveBeenCalledWith('https://numen.md/')
  })
})
