import { describe, expect, it, vi } from 'vitest'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

describe('createPluginHost', () => {
  function createMockDeps(): {
    deps: PluginHostDeps
    registeredKinds: unknown[]
    extensionReaders: Map<string, unknown>
  } {
    const registeredKinds: unknown[] = []
    const extensionReaders = new Map<string, unknown>()

    const deps: PluginHostDeps = {
      registerKind: vi.fn((kind) => {
        registeredKinds.push(kind)
        return () => {
          const index = registeredKinds.indexOf(kind)
          if (index !== -1) registeredKinds.splice(index, 1)
        }
      }),
      registerExtensionReader: vi.fn((ext, reader) => {
        extensionReaders.set(ext, reader)
        return () => {
          extensionReaders.delete(ext)
        }
      }),
      openTab: vi.fn(async () => 'tab-1'),
      getSetting: vi.fn((_pluginId, key) => (key === 'fontSize' ? 14 : undefined)),
      onSettingChange: vi.fn(() => () => {}),
    }

    return { deps, registeredKinds, extensionReaders }
  }

  it('manages readers, tab kinds, commands, and settings', async () => {
    const { deps, registeredKinds, extensionReaders } = createMockDeps()
    const host = createPluginHost(deps)
    const ctx = host.createContext('sample.plugin')

    // Register reader
    const readerSub = ctx.registerReader({
      id: 'custom-pdf',
      fileExtensions: ['.cpdf', 'cpdf2'],
      component: null,
      title: 'Custom PDF',
    })

    expect(deps.registerKind).toHaveBeenCalled()
    expect(extensionReaders.has('.cpdf')).toBe(true)
    expect(extensionReaders.has('cpdf2')).toBe(true)
    expect(host.getRegisteredReaders()).toHaveLength(1)

    // Register tab kind
    const tabSub = ctx.registerTabKind({
      kind: 'my-tab',
      component: null,
      title: 'My Tab',
    })
    expect(host.getRegisteredTabKinds()).toHaveLength(1)
    expect(registeredKinds).toHaveLength(2)

    // Register command
    let executed = false
    const cmdSub = ctx.registerCommand({
      id: 'test.run',
      title: 'Run Test',
      shortcut: 'ctrl+shift+k',
      handler: async () => {
        executed = true
      },
    })
    expect(host.hasCommand('test.run')).toBe(true)
    expect(host.hasCommand('plugin:test.run')).toBe(true)
    expect(host.getRegisteredCommands()).toHaveLength(1)

    // Palette commands
    const palette = host.getPaletteCommands()
    expect(palette).toHaveLength(1)
    expect(palette[0]?.id).toBe('plugin:test.run')

    // Execute command
    await host.executeCommand('plugin:test.run')
    expect(executed).toBe(true)

    // Shortcut matching
    executed = false
    const event = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
      shiftKey: true,
    })
    const handled = host.handleShortcut(event)
    expect(handled).toBe(true)
    expect(executed).toBe(true)

    // Cleanups via dispose
    cmdSub.dispose()
    expect(host.hasCommand('test.run')).toBe(false)

    readerSub.dispose()
    expect(extensionReaders.has('.cpdf')).toBe(false)

    tabSub.dispose()
    expect(host.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('disposes all registrations when plugin is disposed', () => {
    const { deps, extensionReaders } = createMockDeps()
    const host = createPluginHost(deps)
    const ctx = host.createContext('disposable.plugin')

    ctx.registerReader({
      id: 'reader1',
      fileExtensions: ['.ext1'],
      component: null,
    })
    ctx.registerCommand({
      id: 'cmd1',
      title: 'Cmd 1',
      handler: async () => {},
    })

    expect(extensionReaders.has('.ext1')).toBe(true)
    expect(host.hasCommand('cmd1')).toBe(true)

    host.disposePlugin('disposable.plugin')

    expect(extensionReaders.has('.ext1')).toBe(false)
    expect(host.hasCommand('cmd1')).toBe(false)
    expect(host.getRegisteredReaders()).toHaveLength(0)
  })
})
