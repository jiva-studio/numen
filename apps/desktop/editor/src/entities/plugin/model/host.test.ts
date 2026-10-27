import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createPluginHost, createPluginPaneWrapper } from './host'
import { registerPluginStyle } from '@numen/plugin-host'
import { CONVERSATION } from '@numen/plugin-sdk'
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
      getReaderExtensions: vi.fn(
        (_pluginId: string, readerId: string) =>
          ({ 'custom-pdf': ['cpdf', 'cpdf2'], reader1: ['ext1'] })[readerId] ?? [],
      ),
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
      component: null,
      title: 'Custom PDF',
      source: 'book',
    })

    expect(deps.registerKind).toHaveBeenCalled()
    expect(extensionReaders.has('cpdf')).toBe(true)
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
    await host.executeCommand('plugin:test.run', { path: '', title: '', type: 'note' })
    expect(executed).toBe(true)

    // Shortcut matching
    executed = false
    const event = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
      shiftKey: true,
    })
    const handled = host.handleShortcut(event, async () => ({ path: '', title: '', type: 'note' }))
    expect(handled).toBe(true)
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(executed).toBe(true)

    // Cleanups via dispose
    cmdSub.dispose()
    expect(host.hasCommand('test.run')).toBe(false)

    readerSub.dispose()
    expect(extensionReaders.has('cpdf')).toBe(false)

    tabSub.dispose()
    expect(host.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('disposes all registrations when plugin is disposed', () => {
    const { deps, extensionReaders } = createMockDeps()
    const host = createPluginHost(deps)
    const ctx = host.createContext('disposable.plugin')

    ctx.registerReader({
      id: 'reader1',
      component: null,
      source: 'note',
    })
    ctx.registerCommand({
      id: 'cmd1',
      title: 'Cmd 1',
      handler: async () => {},
    })

    expect(extensionReaders.has('ext1')).toBe(true)
    expect(host.hasCommand('cmd1')).toBe(true)

    host.disposePlugin('disposable.plugin')

    expect(extensionReaders.has('ext1')).toBe(false)
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

  it('gives each plugin its own notices and clears them when the plugin is disposed', () => {
    const { deps } = createMockDeps()
    const notices = { show: vi.fn(), setTasks: vi.fn() }
    const getNotices = vi.fn(() => notices)
    const clearNotices = vi.fn()
    const host = createPluginHost({ ...deps, getNotices, clearNotices })

    host.createContext('test.plugin').notices?.show('Saved.')
    host.disposePlugin('test.plugin')

    expect(getNotices).toHaveBeenCalledWith('test.plugin')
    expect(notices.show).toHaveBeenCalledWith('Saved.')
    expect(clearNotices).toHaveBeenCalledWith('test.plugin')
  })

  it('gives a plugin no notices where the window offers none', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(host.createContext('test.plugin').notices).toBeUndefined()
  })

  it('gives each plugin the task host for its own id', () => {
    const { deps } = createMockDeps()
    const tasks = { watch: vi.fn(), watchAll: vi.fn() }
    const getTasks = vi.fn(() => tasks)
    const host = createPluginHost({ ...deps, getTasks })

    expect(host.createContext('test.plugin').tasks).toBe(tasks)
    expect(getTasks).toHaveBeenCalledWith('test.plugin')
  })

  it('gives a plugin no task host where the window offers none', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(host.createContext('test.plugin').tasks).toBeUndefined()
  })

  it('registers context menu, removes it on dispose, and restores on reactivate', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    const ctx1 = host.createContext('sample.plugin')

    ctx1.registerContextMenu({
      id: 'menu-1',
      text: 'Action 1',
      target: 'note',
      handler: async () => {},
    })

    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, kind: 'note' }),
    ).toHaveLength(1)

    // Plugin is deactivated / disposed
    host.disposePlugin('sample.plugin')
    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, kind: 'note' }),
    ).toHaveLength(0)

    // Plugin is reactivated (new context created and registered)
    const ctx2 = host.createContext('sample.plugin')
    ctx2.registerContextMenu({
      id: 'menu-1',
      text: 'Action 1',
      target: 'note',
      handler: async () => {},
    })
    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, kind: 'note' }),
    ).toHaveLength(1)
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

  it('draws a view one plugin declares into the element of another', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    const conversation = defineComponent({
      props: { context: { type: Object, required: true } },
      setup: (props) => () => h('p', (props.context as { focus: string }).focus),
    })
    host
      .createContext('md.numen.chat')
      .registerView({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')

    const mounted = host
      .createContext('md.numen.cards')
      .mountView(CONVERSATION, panel, { focus: 'Decks/Physics.md' })
    expect(panel.textContent).toBe('Decks/Physics.md')

    mounted.dispose()
    expect(panel.textContent).toBe('')
  })

  it('takes a drawn view away with the plugin that declared it', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    const conversation = defineComponent({ setup: () => () => h('p', 'conversation') })
    host
      .createContext('md.numen.chat')
      .registerView({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')
    host.createContext('md.numen.cards').mountView(CONVERSATION, panel, {})

    host.disposePlugin('md.numen.chat')

    expect(panel.textContent).toBe('')
  })
})
