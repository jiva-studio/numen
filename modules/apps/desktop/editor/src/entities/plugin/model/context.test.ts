import { describe, expect, it, vi } from 'vitest'
import { createPluginContext } from './context'
import type { PluginContextHost } from './types'

describe('PluginContext', () => {
  function createMockHost(): { host: PluginContextHost; calls: Record<string, number> } {
    const calls = {
      registerReader: 0,
      registerTabKind: 0,
      registerCommand: 0,
      registerAgentProvider: 0,
      getSetting: 0,
      onSettingChange: 0,
      onVaultChange: 0,
      onFileSelect: 0,
      emitVaultChange: 0,
      emitFileSelect: 0,
    }

    const host: PluginContextHost = {
      registerReader: vi.fn((_id, _reg) => {
        calls.registerReader++
        return { dispose: vi.fn() }
      }),
      registerTabKind: vi.fn((_id, _reg) => {
        calls.registerTabKind++
        return { dispose: vi.fn() }
      }),
      registerCommand: vi.fn((_id, _reg) => {
        calls.registerCommand++
        return { dispose: vi.fn() }
      }),
      registerAgentProvider: vi.fn((_id, _reg) => {
        calls.registerAgentProvider++
        return { dispose: vi.fn() }
      }),
      getSetting: vi.fn((_id: string, key: string) => {
        calls.getSetting++
        return key === 'enabled' ? true : undefined
      }) as unknown as PluginContextHost['getSetting'],
      onSettingChange: vi.fn((_id, _key, _handler) => {
        calls.onSettingChange++
        return { dispose: vi.fn() }
      }),
      onVaultChange: vi.fn((_handler) => {
        calls.onVaultChange++
        return { dispose: vi.fn() }
      }),
      onFileSelect: vi.fn((_handler) => {
        calls.onFileSelect++
        return { dispose: vi.fn() }
      }),
      emitVaultChange: vi.fn((_event) => {
        calls.emitVaultChange++
      }),
      emitFileSelect: vi.fn((_event) => {
        calls.emitFileSelect++
      }),
      openTab: vi.fn(async (_kind, _at) => 'opened-tab-id'),
    }

    return { host, calls }
  }

  it('tracks subscriptions and delegates registrations to host', async () => {
    const { host } = createMockHost()
    const ctx = createPluginContext('test.plugin', host, {
      id: 'test.plugin',
      name: 'Test Plugin',
      version: '1.0.0',
      permissions: ['vault:read', 'vault:write'],
    })

    expect(ctx.pluginId).toBe('test.plugin')
    expect(ctx.subscriptions).toHaveLength(0)

    const sub1 = ctx.registerReader({
      id: 'pdf',
      fileExtensions: ['.pdf'],
      component: null,
    })
    expect(ctx.subscriptions).toHaveLength(1)
    expect(host.registerReader).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ id: 'pdf' }),
    )

    const sub2 = ctx.registerCommand({
      id: 'cmd1',
      title: 'Command 1',
      handler: async () => {},
    })
    expect(ctx.subscriptions).toHaveLength(2)
    expect(host.registerCommand).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ id: 'cmd1' }),
    )

    const sub3 = ctx.registerTabKind({
      kind: 'custom-tab',
      component: null,
    })
    expect(ctx.subscriptions).toHaveLength(3)
    expect(host.registerTabKind).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ kind: 'custom-tab' }),
    )

    const sub4 = ctx.registerAgentProvider({
      id: 'agent1',
      name: 'Agent 1',
      createAgent: vi.fn(),
    })
    expect(ctx.subscriptions).toHaveLength(4)
    expect(host.registerAgentProvider).toHaveBeenCalledWith(
      'test.plugin',
      expect.objectContaining({ id: 'agent1' }),
    )

    expect(ctx.getSetting('enabled')).toBe(true)
    expect(host.getSetting).toHaveBeenCalledWith('test.plugin', 'enabled')

    const sub5 = ctx.onSettingChange('enabled', () => {})
    expect(ctx.subscriptions).toHaveLength(5)
    expect(host.onSettingChange).toHaveBeenCalledWith(
      'test.plugin',
      'enabled',
      expect.any(Function),
    )

    const sub6 = ctx.onVaultChange(() => {})
    expect(ctx.subscriptions).toHaveLength(6)

    const sub7 = ctx.onFileSelect(() => {})
    expect(ctx.subscriptions).toHaveLength(7)

    ctx.emitVaultChange({ vaultPath: '/path' })
    expect(host.emitVaultChange).toHaveBeenCalledWith({ vaultPath: '/path' })

    ctx.emitFileSelect({ path: '/file.md' })
    expect(host.emitFileSelect).toHaveBeenCalledWith({ path: '/file.md' })

    const opened = await ctx.openTab?.('url', 'https://example.com')
    expect(opened).toBe('opened-tab-id')
    expect(host.openTab).toHaveBeenCalledWith('url', 'https://example.com')

    sub1.dispose()
    sub2.dispose()
    sub3.dispose()
    sub4.dispose()
    sub5.dispose()
    sub6.dispose()
    sub7.dispose()
  })

  it('enforces vault:read permission for onVaultChange, onFileSelect, and emitFileSelect', () => {
    const { host } = createMockHost()
    const ctxNoPerms = createPluginContext('restricted.plugin', host, {
      id: 'restricted.plugin',
      name: 'Restricted',
      version: '1.0.0',
      permissions: [],
    })

    expect(() => ctxNoPerms.onVaultChange(() => {})).toThrowError(/vault:read/)
    expect(() => ctxNoPerms.onFileSelect(() => {})).toThrowError(/vault:read/)
    expect(() => ctxNoPerms.emitFileSelect({ path: '/test.md' })).toThrowError(/vault:read/)

    const ctxWithRead = createPluginContext('reader.plugin', host, {
      id: 'reader.plugin',
      name: 'Reader',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(() => ctxWithRead.onVaultChange(() => {})).not.toThrow()
    expect(() => ctxWithRead.onFileSelect(() => {})).not.toThrow()
    expect(() => ctxWithRead.emitFileSelect({ path: '/test.md' })).not.toThrow()
  })

  it('enforces vault:write permission for emitVaultChange', () => {
    const { host } = createMockHost()
    const ctxReadOnly = createPluginContext('readonly.plugin', host, {
      id: 'readonly.plugin',
      name: 'Read Only',
      version: '1.0.0',
      permissions: ['vault:read'],
    })

    expect(() => ctxReadOnly.emitVaultChange({ vaultPath: '/test' })).toThrowError(/vault:write/)

    const ctxWrite = createPluginContext('writer.plugin', host, {
      id: 'writer.plugin',
      name: 'Writer',
      version: '1.0.0',
      permissions: ['vault:write'],
    })

    expect(() => ctxWrite.emitVaultChange({ vaultPath: '/test' })).not.toThrow()
    expect(host.emitVaultChange).toHaveBeenCalledWith({ vaultPath: '/test' })
  })
})
