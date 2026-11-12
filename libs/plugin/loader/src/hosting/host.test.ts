import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref, type PropType } from 'vue'
import { mount } from '@vue/test-utils'
import { createPluginHost, createPluginPaneWrapper } from './host'
import { registerPluginStyle } from '../styles'
import {
  CONVERSATION,
  NOTE_EDITOR,
  type NoteEditorContext,
  type VaultNotes,
} from '@numen/plugin-sdk'
import { createMockBackgroundTaskHost } from '@numen/plugin-sdk/testing'
import { createMockManifest } from '@numen/core-client/testing'
import type { PluginHostDeps } from './types'
import { STUB_AGENT, STUB_COMPONENT } from '../testing/stubComponent'

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
        },
      ],
      target: 'desktop.editor',
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
    host.keybindings.registerKeybindings('sample.plugin', [
      { key: 'ctrl+shift+k', command: 'test.run' },
    ])
    const ctx = host.lifecycle.createContext(
      'sample.plugin',
      createMockManifest('sample.plugin', {
        name: 'Sample',
        permissions: ['vault:read'],
        readers: ['custom-pdf'],
        tabKinds: ['my-tab'],
        commands: ['test.run'],
      }),
    )

    // Register reader
    const readerSub = ctx.register.reader({
      id: 'custom-pdf',
      component: STUB_COMPONENT,
      title: 'Custom PDF',
    })

    expect(deps.registerKind).toHaveBeenCalled()
    expect(extensionReaders.has('cpdf')).toBe(true)
    expect(extensionReaders.has('cpdf2')).toBe(true)
    expect(host.registered.getRegisteredReaders()).toHaveLength(1)

    // Register tab kind
    const tabSub = ctx.register.tabKind({
      kind: 'my-tab',
      component: STUB_COMPONENT,
      title: 'My Tab',
    })
    expect(host.registered.getRegisteredTabKinds()).toHaveLength(1)
    expect(registeredKinds).toHaveLength(2)

    // Register command
    let executed = false
    const cmdSub = ctx.register.command({
      id: 'test.run',
      handler: async () => {
        executed = true
      },
    })
    expect(host.commands.hasCommand('sample.plugin:test.run')).toBe(true)
    expect(host.registered.getRegisteredCommands()).toHaveLength(1)

    // Palette commands
    const palette = host.commands.getPaletteCommands()
    expect(palette).toHaveLength(1)
    expect(palette[0]?.id).toBe('sample.plugin:test.run')

    // Execute command
    await host.commands.executeCommand('sample.plugin:test.run', {
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
    const handled = host.commands.handleShortcut(event, async () => ({
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
    expect(host.commands.hasCommand('sample.plugin:test.run')).toBe(false)

    readerSub.dispose()
    expect(extensionReaders.has('cpdf')).toBe(false)

    tabSub.dispose()
    expect(host.registered.getRegisteredTabKinds()).toHaveLength(0)
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
    const ctx = host.lifecycle.createContext(
      'disposable.plugin',
      createMockManifest('disposable.plugin', { readers: ['reader1'], commands: ['cmd1'] }),
    )

    ctx.register.reader({
      id: 'reader1',
      component: STUB_COMPONENT,
    })
    ctx.register.command({
      id: 'cmd1',
      handler: vi.fn(() => Promise.resolve()),
    })

    expect(extensionReaders.has('ext1')).toBe(true)
    expect(host.commands.hasCommand('disposable.plugin:cmd1')).toBe(true)

    host.lifecycle.disposePlugin('disposable.plugin')

    expect(extensionReaders.has('ext1')).toBe(false)
    expect(host.commands.hasCommand('disposable.plugin:cmd1')).toBe(false)
    expect(host.registered.getRegisteredReaders()).toHaveLength(0)
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
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader,
      openTab: vi.fn(async () => 't1'),
      registerEditor: vi.fn(() => undefined as unknown as () => void),
    })
    const ctxNoExt = hostNoExt.lifecycle.createContext(
      'noext.plugin',
      createMockManifest('noext.plugin', {
        name: 'No ext',
        permissions: ['vault:read'],
        readers: ['r-noext'],
        tabKinds: ['void-editor'],
      }),
    )
    ctxNoExt.register.reader({ id: 'r-noext', component: STUB_COMPONENT })
    expect(registerExtensionReader).not.toHaveBeenCalled()
    const tabSubVoid = ctxNoExt.register.tabKind({ kind: 'void-editor', component: STUB_COMPONENT })
    expect(() => tabSubVoid.dispose()).not.toThrow()

    // Host with getReaderExtensions and getTabState returning undefined vs object
    const hostWithExt = createPluginHost({
      target: 'desktop.editor',
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader,
      getReaderExtensions: () => ['pdf'],
      openTab: vi.fn(async () => 't1'),
      getTabState: (id) => stateMap.get(id),
      onError,
    })
    hostWithExt.lifecycle
      .createContext('ext.plugin', createMockManifest('ext.plugin', { readers: ['r-pdf'] }))
      .register.reader({ id: 'r-pdf', component: STUB_COMPONENT })
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

    host.lifecycle.disposePlugin('test.plugin')
    expect(closeTabsForPlugin).toHaveBeenCalledWith('test.plugin')
  })

  it('gives each plugin its own notices and clears them when the plugin is disposed', () => {
    const { deps } = createMockDeps()
    const notices = { show: vi.fn(), setTasks: vi.fn() }
    const getNotices = vi.fn(() => notices)
    const clearNotices = vi.fn()
    const host = createPluginHost({ ...deps, getNotices, clearNotices })

    host.lifecycle.createContext('test.plugin').ui.notices.show('Saved.')
    host.lifecycle.disposePlugin('test.plugin')

    expect(getNotices).toHaveBeenCalledWith('test.plugin')
    expect(notices.show).toHaveBeenCalledWith('Saved.')
    expect(clearNotices).toHaveBeenCalledWith('test.plugin')
  })

  it('drops what a plugin reports where the window offers no notices', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(() =>
      host.lifecycle.createContext('test.plugin').ui.notices.show('Saved.'),
    ).not.toThrow()
  })

  it('gives each plugin the task host for its own id', () => {
    const { deps } = createMockDeps()
    const tasks = { watch: vi.fn(), watchAll: vi.fn() }
    const getTasks = vi.fn(() => tasks)
    const host = createPluginHost({ ...deps, getTasks })

    const handler = vi.fn()
    host.lifecycle.createContext('test.plugin').work.tasks?.watch(handler)

    expect(tasks.watch).toHaveBeenCalledWith(handler)
    expect(getTasks).toHaveBeenCalledWith('test.plugin')
  })

  it('gives a plugin no task host where the window offers none', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(host.lifecycle.createContext('test.plugin').work.tasks).toBeUndefined()
  })

  it('gives every plugin the background tasks of the window', () => {
    const { deps } = createMockDeps()
    const backgroundTasks = createMockBackgroundTaskHost()
    const host = createPluginHost({ ...deps, getBackgroundTasks: () => backgroundTasks })

    const first = vi.fn()
    const second = vi.fn()
    host.lifecycle.createContext('test.plugin').work.backgroundTasks?.watch(first)
    host.lifecycle.createContext('other.plugin').work.backgroundTasks?.watch(second)

    expect(first).toHaveBeenCalled()
    expect(second).toHaveBeenCalled()
    expect(backgroundTasks.watch).toBeDefined()
  })

  it('gives a plugin no background tasks where the window offers none', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost({ ...deps, getBackgroundTasks: () => undefined })

    expect(host.lifecycle.createContext('test.plugin').work.backgroundTasks).toBeUndefined()
    expect(
      createPluginHost(deps).lifecycle.createContext('test.plugin').work.backgroundTasks,
    ).toBeUndefined()
  })

  it('files the health items of a plugin under its own id', async () => {
    const { deps } = createMockDeps()
    const filed: string[] = []
    const host = createPluginHost({
      ...deps,
      health: {
        reportHealth: (pluginId, item) => {
          filed.push(`${pluginId} ${item.key}`)
          return Promise.resolve()
        },
        clearHealth: (pluginId, key) => {
          filed.push(`${pluginId} -${key}`)
          return Promise.resolve()
        },
      },
    })

    const health = host.lifecycle.createContext('org.example.atlas').health
    await health?.report({ key: 'connection', severity: 'error', text: 'Choose a connection' })
    await health?.clear('connection')

    expect(filed).toStrictEqual(['org.example.atlas connection', 'org.example.atlas -connection'])
    expect(createPluginHost(deps).lifecycle.createContext('org.example.atlas').health).toBe(
      undefined,
    )
  })

  it('delegates getSetting and onSettingChange through host', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)

    expect(host.lifecycle.getSetting('test.plugin', 'fontSize')).toBe(14)
    expect(deps.getSetting).toHaveBeenCalledWith('test.plugin', 'fontSize')

    const handler = vi.fn()
    const sub = host.lifecycle.onSettingChange('test.plugin', 'fontSize', handler)
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
    host.lifecycle.createContext('test.plugin', initialManifest)

    expect(() => {
      host.lifecycle.registerManifest(
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
    host.lifecycle
      .createContext(
        'md.numen.chat',
        createMockManifest('md.numen.chat', {
          permissions: ['vault:read', 'vault:write'],
          viewCapabilities: [CONVERSATION],
        }),
      )
      .register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')

    const mounted = host.lifecycle
      .createContext('org.example.sample')
      .ui.mountView(CONVERSATION, panel, { agent: STUB_AGENT, focus: 'Samples/Physics.md' })
    expect(panel.textContent).toBe('Samples/Physics.md')

    mounted.dispose()
    expect(panel.textContent).toBe('')
  })

  it('takes a drawn view away with the plugin that declared it', () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    const conversation = defineComponent({ setup: () => () => h('p', 'conversation') })
    host.lifecycle
      .createContext(
        'md.numen.chat',
        createMockManifest('md.numen.chat', {
          permissions: ['vault:read', 'vault:write'],
          viewCapabilities: [CONVERSATION],
        }),
      )
      .register.view({ capability: CONVERSATION, component: conversation })
    const panel = document.createElement('section')
    host.lifecycle
      .createContext('org.example.sample')
      .ui.mountView(CONVERSATION, panel, { agent: STUB_AGENT, focus: '' })

    host.lifecycle.disposePlugin('md.numen.chat')

    expect(panel.textContent).toBe('')
  })

  it('routes a drop to the handler another plugin registered until it is disposed', async () => {
    const { deps } = createMockDeps()
    const host = createPluginHost(deps)
    host.lifecycle
      .createContext(
        'md.numen.links',
        createMockManifest('md.numen.links', {
          permissions: ['vault:read'],
          dropTypes: ['text/uri-list'],
        }),
      )
      .register.dropHandler?.({
        types: ['text/uri-list'],
        drop: async (folder, data) => [`${folder}/${data['text/uri-list']}.md`],
      })
    const drops = host.lifecycle.createContext(
      'md.numen.views.file.tree',
      createMockManifest('md.numen.views.file.tree', {
        name: 'Tree',
        permissions: ['vault:write'],
      }),
    ).ui.drops

    expect(drops?.canDrop(['text/uri-list'])).toBe(true)
    expect(await drops?.drop('talks', { 'text/uri-list': 'clip' })).toEqual(['talks/clip.md'])

    host.lifecycle.disposePlugin('md.numen.links')
    expect(drops?.canDrop(['text/uri-list'])).toBe(false)
  })
})

describe('a view mounted by a plugin', () => {
  // The note editor view of the notes plugin, reduced: like the real one it reads the note of
  // context.path unconditionally, through the notes service of the plugin that registered it.
  const createHost = (registerNow = true) => {
    const read = vi.fn(async (path: string) => ({ body: `Body of ${path}` }))
    const host = createPluginHost({
      target: 'desktop.editor',
      registerKind: () => vi.fn(),
      registerExtensionReader: () => vi.fn(),
      openTab: () => Promise.resolve('tab'),
      getNotes: () => ({ read }) as unknown as VaultNotes,
    } as unknown as PluginHostDeps)
    const notesPlugin = host.lifecycle.createContext(
      'md.numen.notes',
      createMockManifest('md.numen.notes', {
        name: 'Notes',
        permissions: ['vault:read', 'vault:write'],
        viewCapabilities: [NOTE_EDITOR, CONVERSATION],
      }),
    )
    const registerViews = () => {
      notesPlugin.register.view({
        capability: NOTE_EDITOR,
        component: defineComponent({
          props: { context: { type: Object as PropType<NoteEditorContext>, required: true } },
          setup(props) {
            const body = ref('')
            notesPlugin.vault.notes!.read(props.context.path).then(
              (one) => (body.value = one.body),
              (error: unknown) => (body.value = `Failed: ${String(error)}`),
            )
            return () => h('div', body.value)
          },
        }),
      })
      notesPlugin.register.view({
        capability: CONVERSATION,
        component: defineComponent({ setup: () => () => h('p', 'conversation') }),
      })
    }
    if (registerNow) registerViews()
    const createCaller = (permissions: string[]) =>
      host.lifecycle.createContext(
        'md.numen.caller',
        createMockManifest('md.numen.caller', { name: 'Caller', permissions }),
      )
    return { read, createCaller, registerViews }
  }
  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve, 20))
    await nextTick()
  }

  it('refuses to mount a view with a note path to a plugin without vault:read', async () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')

    expect(() =>
      createCaller([]).ui.mountView(NOTE_EDITOR, element, { path: 'Notes/Example.md' }),
    ).toThrow(/vault:read/)
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount the note editor with an empty context to a plugin without vault:read', async () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')

    expect(() => createCaller([]).ui.mountView(NOTE_EDITOR, element, { path: '' })).toThrow(
      /vault:read/,
    )
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount the note editor with a path that reads falsy first and set afterwards', async () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')
    let reads = 0
    const context: NoteEditorContext = {
      get path() {
        return reads++ === 0 ? '' : 'Notes/Example.md'
      },
    }

    expect(() => createCaller([]).ui.mountView(NOTE_EDITOR, element, context)).toThrow(/vault:read/)
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount the note editor with a context that hides its path from in', async () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')
    const context = new Proxy({} as NoteEditorContext, {
      has: () => false,
      get: (_target, key) => (key === 'path' ? 'Notes/Example.md' : undefined),
    })

    expect(() => createCaller([]).ui.mountView(NOTE_EDITOR, element, context)).toThrow(/vault:read/)
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount the note editor whose path is set after the mount, before the view exists', async () => {
    const { read, createCaller, registerViews } = createHost(false)
    const element = document.createElement('div')
    const context: { path?: string } = {}

    expect(() =>
      createCaller([]).ui.mountView(NOTE_EDITOR, element, context as NoteEditorContext),
    ).toThrow(/vault:read/)
    context.path = 'Notes/Example.md'
    registerViews()
    await settle()

    expect(read).not.toHaveBeenCalled()
    expect(element.textContent).toBe('')
  })

  it('refuses to mount the note editor to a plugin that holds vault:read and not vault:write', () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')

    expect(() =>
      createCaller(['vault:read']).ui.mountView(NOTE_EDITOR, element, { path: 'Notes/Example.md' }),
    ).toThrow(/vault:write/)
    expect(read).not.toHaveBeenCalled()
  })

  it('mounts a view with a note path for a plugin with vault:read and vault:write', async () => {
    const { createCaller } = createHost()
    const element = document.createElement('div')

    createCaller(['vault:read', 'vault:write']).ui.mountView(NOTE_EDITOR, element, {
      path: 'Notes/Example.md',
    })
    await vi.waitFor(() => expect(element.textContent).toBe('Body of Notes/Example.md'))
  })

  it('mounts the conversation view for a plugin without vault:read', async () => {
    const { read, createCaller } = createHost()
    const element = document.createElement('div')

    expect(() =>
      createCaller([]).ui.mountView(CONVERSATION, element, { agent: STUB_AGENT, focus: '' }),
    ).not.toThrow()
    await settle()

    expect(element.textContent).toBe('conversation')
    expect(read).not.toHaveBeenCalled()
  })
})
