import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createPluginHost, createPluginPaneWrapper } from './host'
import { registerPluginStyle } from './styles'
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
      source: 'book',
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
      source: 'note',
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

  it('calls closeTabsForPlugin when plugin is disposed', () => {
    const { deps } = createMockDeps()
    const closeTabsForPlugin = vi.fn()
    const host = createPluginHost({ ...deps, closeTabsForPlugin })

    host.disposePlugin('test.plugin')
    expect(closeTabsForPlugin).toHaveBeenCalledWith('test.plugin')
  })

  it('delegates getSetting and onSettingChange through host', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(host.getSetting('test.plugin', 'fontSize')).toBe(14)
    expect(deps.getSetting).toHaveBeenCalledWith('test.plugin', 'fontSize')

    const handler = vi.fn()
    const sub = host.onSettingChange('test.plugin', 'fontSize', handler)
    expect(deps.onSettingChange).toHaveBeenCalledWith(
      'test.plugin',
      'fontSize',
      expect.any(Function),
    )
    expect(sub).toHaveProperty('dispose')
  })

  it('rejects manifest registration when context already exists', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    const initialManifest = {
      id: 'test.plugin',
      name: 'Original',
      version: '1.0.0',
      permissions: [],
    }
    host.createContext('test.plugin', initialManifest)

    expect(() => {
      host.registerManifest('test.plugin', {
        id: 'test.plugin',
        name: 'Tampered',
        version: '2.0.0',
        permissions: ['vault:read'],
      })
    }).toThrow('Cannot replace the manifest for plugin "test.plugin" while its context is cached')
  })

  it('isolates crashing plugin components with error boundary fallback', async () => {
    const CrashingComponent = defineComponent({
      name: 'CrashingComponent',
      setup() {
        return () => {
          throw new Error('Simulated plugin component render crash')
        }
      },
    })

    const Wrapper = createPluginPaneWrapper('crashing.plugin', CrashingComponent)
    const wrapper = mount(Wrapper, {
      props: { state: { path: '/test/note.md' } },
    })
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.plugin-error-fallback').exists()).toBe(true)
    expect(wrapper.text()).toContain('Plugin "crashing.plugin" crashed')
    expect(wrapper.text()).toContain('Simulated plugin component render crash')
  })

  it('encapsulates plugin view and stylesheet within Shadow DOM container', async () => {
    const SampleComponent = defineComponent({
      name: 'SampleComponent',
      setup() {
        return () => h('div', { class: 'sample-content' }, 'Plugin Content')
      },
    })

    registerPluginStyle('sample.plugin', '/api/plugins/sample.plugin/styles.css')
    const Wrapper = createPluginPaneWrapper('sample.plugin', SampleComponent)
    const wrapper = mount(Wrapper, {
      props: { state: {} },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    const hostDiv = wrapper.element as HTMLElement
    expect(hostDiv.classList.contains('plugin-pane-wrapper')).toBe(true)
    expect(hostDiv.getAttribute('data-plugin-id')).toBe('sample.plugin')
    expect(hostDiv.shadowRoot).not.toBeNull()

    const shadow = hostDiv.shadowRoot!
    const link = shadow.querySelector('link[rel="stylesheet"]')
    expect(link).not.toBeNull()
    expect(link?.getAttribute('href')).toBe('/api/plugins/sample.plugin/styles.css')
    expect(shadow.querySelector('.sample-content')?.textContent).toBe('Plugin Content')

    expect(document.head.querySelector('link[data-plugin-id="sample.plugin"]')).toBeNull()

    wrapper.unmount()
  })
})
