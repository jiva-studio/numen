import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createPluginHost, createPluginPaneWrapper } from './host'
import { disposeAllEntries, type HostRegistry } from './registry'
import { createStatusViewRegistry } from '../statusViews'
import { registerPluginStyle } from '../styles'
import { CONVERSATION } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
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
      getCommandDeclarations: () => [
        {
          pluginId: 'sample.plugin',
          id: 'test.run',
          title: 'Run Test',
          shortcut: 'ctrl+shift+k',
        },
      ],
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
      onSettingChange: vi.fn(() => vi.fn()),
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
      handler: async () => {
        executed = true
      },
    })
    expect(host.hasCommand('sample.plugin:test.run')).toBe(true)
    expect(host.getRegisteredCommands()).toHaveLength(1)

    // Palette commands
    const palette = host.getPaletteCommands()
    expect(palette).toHaveLength(1)
    expect(palette[0]?.id).toBe('sample.plugin:test.run')

    // Execute command
    await host.executeCommand('sample.plugin:test.run', {
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })
    expect(executed).toBe(true)

    // Shortcut matching
    executed = false
    const event = new KeyboardEvent('keydown', {
      key: 'k',
      ctrlKey: true,
      shiftKey: true,
    })
    const handled = host.handleShortcut(event, async () => ({
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    }))
    expect(handled).toBe(true)
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(executed).toBe(true)

    // Cleanups via dispose
    cmdSub.dispose()
    expect(host.hasCommand('sample.plugin:test.run')).toBe(false)

    readerSub.dispose()
    expect(extensionReaders.has('cpdf')).toBe(false)

    tabSub.dispose()
    expect(host.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('disposes all registrations when plugin is disposed', () => {
    const { deps, extensionReaders } = createMockDeps()
    const host = createPluginHost({
      ...deps,
      getCommandDeclarations: () => [
        {
          pluginId: 'disposable.plugin',
          id: 'cmd1',
          title: 'Cmd 1',
        },
      ],
    })
    const ctx = host.createContext('disposable.plugin')

    ctx.registerReader({
      id: 'reader1',
      component: null,
    })
    ctx.registerCommand({
      id: 'cmd1',
      handler: vi.fn(() => Promise.resolve()),
    })

    expect(extensionReaders.has('ext1')).toBe(true)
    expect(host.hasCommand('disposable.plugin:cmd1')).toBe(true)

    host.disposePlugin('disposable.plugin')

    expect(extensionReaders.has('ext1')).toBe(false)
    expect(host.hasCommand('disposable.plugin:cmd1')).toBe(false)
    expect(host.getRegisteredReaders()).toHaveLength(0)

    // Disposing a plugin cleans up its agent providers, and disposeAllEntries cleans up agentProviders without context
    const reg: HostRegistry = {
      vaultListeners: new Set(),
      fileListeners: new Set(),
      readers: new Map(),
      tabKinds: new Map(),
      commands: { value: new Map() },
      contextMenus: { value: new Map() },
      settingsViews: { value: new Map() },
      agentProviders: new Map([
        [
          'ap-orphan',
          {
            pluginId: 'orphan.plugin',
            reg: { id: 'ap-orphan', name: 'Orphan', createAgent: vi.fn() },
          },
        ],
      ]),
      contexts: new Map(),
      manifests: new Map(),
      activated: { value: new Set() },
      disposed: new Set(),
      reportedIssues: new Set(),
    }
    disposeAllEntries({ registry: reg, statusViews: createStatusViewRegistry(), deps })
    expect(reg.agentProviders.size).toBe(0)
  })

  it('handles reader extensions fallback, missing tab state on open, and non-function registerEditor return', async () => {
    const onError = vi.fn()
    let extCallback: ((path: string, selector?: Record<string, unknown>) => void) | undefined
    const stateMap = new Map<string, Record<string, unknown>>()
    const registerExtensionReader = vi.fn((_ext, cb) => {
      extCallback = cb
      return vi.fn()
    })

    // Host without getReaderExtensions does not register extension readers
    const hostNoExt = createPluginHost({
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader,
      openTab: vi.fn(async () => 't1'),
      registerEditor: vi.fn(() => undefined as unknown as () => void),
    })
    const ctxNoExt = hostNoExt.createContext('noext.plugin')
    ctxNoExt.registerReader({ id: 'r-noext', component: null })
    expect(registerExtensionReader).not.toHaveBeenCalled()
    const tabSubVoid = ctxNoExt.registerTabKind({ kind: 'void-editor', component: null })
    expect(() => tabSubVoid.dispose()).not.toThrow()

    // Host with getReaderExtensions and getTabState returning undefined vs object
    const hostWithExt = createPluginHost({
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader,
      getReaderExtensions: () => ['pdf'],
      openTab: vi.fn(async () => 't1'),
      getTabState: (id) => stateMap.get(id),
      onError,
    })
    hostWithExt.createContext('ext.plugin').registerReader({ id: 'r-pdf', component: null })
    expect(extCallback).toBeDefined()

    // When getTabState returns undefined, selector assignment is skipped without error
    extCallback!('doc.pdf', { page: 5 })
    await new Promise((r) => setTimeout(r, 0))
    expect(onError).not.toHaveBeenCalled()

    // When getTabState returns an object, selector is copied onto it
    const tabState: Record<string, unknown> = {}
    stateMap.set('t1', tabState)
    const sel = { page: 5 }
    extCallback!('doc.pdf', sel)
    await new Promise((r) => setTimeout(r, 0))
    expect(tabState.selector).toEqual({ page: 5 })
    expect(tabState.selector).not.toBe(sel)
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
      handler: vi.fn(() => Promise.resolve()),
    })

    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, isNote: true }),
    ).toHaveLength(1)

    // Plugin is deactivated / disposed
    host.disposePlugin('sample.plugin')
    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, isNote: true }),
    ).toHaveLength(0)

    // Plugin is reactivated (new context created and registered)
    const ctx2 = host.createContext('sample.plugin')
    ctx2.registerContextMenu({
      id: 'menu-1',
      text: 'Action 1',
      target: 'note',
      handler: vi.fn(() => Promise.resolve()),
    })
    expect(
      host.getContextMenuItems({ path: 'test.md', isFolder: false, isNote: true }),
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
    const initialManifest = createMockManifest('test.plugin', { name: 'Original' })
    host.createContext('test.plugin', initialManifest)

    expect(() => {
      host.registerManifest(
        'test.plugin',
        createMockManifest('test.plugin', {
          name: 'Tampered',
          version: '2.0.0',
          permissions: ['vault:read'],
        }),
      )
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

    registerPluginStyle('sample.plugin', '/plugin/sample.plugin/styles.css')
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
    expect(link?.getAttribute('href')).toBe('/plugin/sample.plugin/styles.css')
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
      .createContext('org.example.sample')
      .mountView(CONVERSATION, panel, { focus: 'Samples/Physics.md' })
    expect(panel.textContent).toBe('Samples/Physics.md')

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
    host.createContext('org.example.sample').mountView(CONVERSATION, panel, {})

    host.disposePlugin('md.numen.chat')

    expect(panel.textContent).toBe('')
  })

  it('routes a drop to the handler another plugin registered until it is disposed', async () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    host.createContext('md.numen.links').registerDropHandler?.({
      types: ['text/uri-list'],
      drop: async (folder, data) => [`${folder}/${data['text/uri-list']}.md`],
    })
    const drops = host.createContext('md.numen.views.file.tree').drops

    expect(drops?.canDrop(['text/uri-list'])).toBe(true)
    expect(await drops?.drop('talks', { 'text/uri-list': 'clip' })).toEqual(['talks/clip.md'])

    host.disposePlugin('md.numen.links')
    expect(drops?.canDrop(['text/uri-list'])).toBe(false)
  })
})
