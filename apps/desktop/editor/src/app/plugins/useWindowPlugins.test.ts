import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import {
  DEFAULT_PLUGIN_ENABLED,
  type AgentHost,
  type PluginContext,
  type WorkspaceHost,
} from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { createTestTabs } from '@/testing/tabs'
import type { AnyTabKind } from '@/entities/tab'
import type { Manifest } from '@numen/core-client'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginSupport, getFrontendPluginIds, watchPluginSettings } from './useWindowPlugins'
import { getNoteIcon } from '@/entities/note'
import { createBackendHandlers } from './commands/backendHandlers'

const createSupport = () => {
  const windowTabs = createTestTabs()
  const support = createPluginSupport({
    windowTabs,
    tabLinks: {} as never,
    tabOpeners: {
      registerExtensionReader: vi.fn(() => () => undefined),
      registerEditor: vi.fn(() => () => undefined),
    } as never,
    settingsHolder: {},
    services: {
      workspace: { getNoteInFocus: () => ({ path: '', title: '' }) } as unknown as WorkspaceHost,
      agent: { finish: () => Promise.resolve(), getModelsAt: () => [] } as unknown as AgentHost,
    },
  })
  return { windowTabs, support }
}

const coreKind = (state: object): AnyTabKind => ({
  kind: 'core',
  open: () => state,
  pane: {},
})

describe('the platform plugins read', () => {
  it('is Windows where the window gives none, as the frame reports without the runtime', () => {
    const { support } = createSupport()

    expect(support.pluginHost.createContext('md.numen.a').app.platform).toBe('windows')
  })
})

describe('closing the tabs of a plugin', () => {
  it('replaces the tabs of the kinds the plugin registered with placeholders', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost
      .createContext('md.numen.a')
      .registerTabKind({ kind: 'a-kind', component: null, title: 'A' })
    const id = await windowTabs.createTab('a-kind', 'x')
    windowTabs.show(id)

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)?.state).toMatchObject({
      kind: 'a-kind',
      targetPath: 'x',
      saved: undefined,
      title: 'x',
    })
  })

  it('keeps a tab of a core kind whose state names the plugin', async () => {
    const { windowTabs, support } = createSupport()
    windowTabs.registerKind(coreKind({ pluginId: 'md.numen.a' }))
    const id = await windowTabs.createTab('core', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)).not.toBeNull()
  })

  it('keeps the tabs of another plugin', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost
      .createContext('md.numen.b')
      .registerTabKind({ kind: 'b-kind', component: null, title: 'B' })
    const id = await windowTabs.createTab('b-kind', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)).not.toBeNull()
  })
})

/** A plugin that adds one tab kind to the window. */
const manifest: Manifest = createMockManifest('md.numen.sample.chat', { name: 'Chat' })

const activate = (context: PluginContext) => {
  context.registerTabKind({ kind: 'chat', component: null, title: 'Chat' })
}

describe('the tab kind a plugin adds', () => {
  const activateChat = async () => {
    const { windowTabs, support } = createSupport()
    activate(support.pluginHost.createContext(manifest.id, manifest))
    return { windowTabs, support }
  }

  it('is a kind of the window while the plugin is active', async () => {
    const { windowTabs } = await activateChat()

    expect(windowTabs.hasKind('chat')).toBe(true)
    expect(windowTabs.getTab(await windowTabs.createTab('chat', ''))).not.toBeNull()
  })

  it('goes, with its open tabs turned into placeholders, when the plugin is disabled', async () => {
    const { windowTabs, support } = await activateChat()
    const id = await windowTabs.createTab('chat', '')

    support.pluginHost.disposePlugin(manifest.id)

    expect(windowTabs.hasKind('chat')).toBe(false)
    expect(windowTabs.getTab(id)?.state).toMatchObject({
      kind: 'chat',
      targetPath: '',
      saved: undefined,
      title: 'Chat',
    })
  })
})

describe('the settings and notices a plugin is given', () => {
  const createWith = (more: Partial<Parameters<typeof createPluginSupport>[0]>) =>
    createPluginSupport({
      windowTabs: createTestTabs(),
      tabLinks: {} as never,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      settingsHolder: {},
      ...more,
    })

  it('reads a setting of the plugin, and the default of its switch where none is written', () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = { getCommittedSetting: (path: readonly string[]) => settings.value[path[1] ?? ''] }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const ctx = pluginHost.createContext('md.numen.a')

    expect(ctx.getSetting('fontSize')).toBe(14)
    expect(ctx.getSetting('enabled')).toBe(DEFAULT_PLUGIN_ENABLED)
    expect(ctx.getSetting('missing')).toBeUndefined()
  })

  it('prefers the committed setting over an unconfirmed pending write', () => {
    const rest = {
      getCommittedSetting: (path: readonly string[]) => (path[1] === 'enabled' ? false : undefined),
      getSetting: () => true,
    }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const ctx = pluginHost.createContext('md.numen.a')

    expect(ctx.getSetting('enabled')).toBe(false)
  })

  it('registers the extensions declared for a reader in the plugin settings', () => {
    const registerExtensionReader = vi.fn(() => () => undefined)
    const rest = {
      plugins: ref([
        {
          id: 'md.numen.other',
          readers: [{ id: 'document', extensions: ['.other'] }],
        },
        {
          id: 'md.numen.pdf',
          readers: [
            { id: 'book', extensions: ['.epub'] },
            { id: 'document', extensions: ['.pdf'] },
          ],
        },
      ]),
    }
    const { pluginHost } = createWith({
      tabOpeners: {
        registerExtensionReader,
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      settingsHolder: { current: { rest } as never },
    })

    pluginHost
      .createContext('md.numen.pdf')
      .registerReader({ id: 'document', title: 'Document', component: null })

    expect(registerExtensionReader).toHaveBeenCalledTimes(1)
    expect(registerExtensionReader).toHaveBeenCalledWith('.pdf', expect.any(Function), undefined)
  })

  it('lists the declared commands of a plugin with a frontend while it loads, and none of a plugin without one', () => {
    const rest = {
      plugins: ref([
        { id: 'md.numen.tree', hasFrontend: true },
        { id: 'md.numen.job', hasFrontend: false },
      ]),
    }
    const { pluginHost } = createWith({
      settingsHolder: { current: { rest } as never },
      getCommandDeclarations: () => [
        { pluginId: 'md.numen.tree', id: 'tree.open', title: 'Open tree' },
        { pluginId: 'md.numen.job', id: 'job.run', title: 'Run job' },
      ],
    })

    expect(pluginHost.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.numen.tree:tree.open',
    ])
  })

  it('reports a declared command with no handler for a plugin without a backend, and for none with one', () => {
    const rest = {
      plugins: ref([
        { id: 'md.numen.tree', hasFrontend: true, hasBackend: false },
        { id: 'md.numen.cards', hasFrontend: true, hasBackend: true },
      ]),
    }
    const reportCommandIssue = vi.fn()
    const { pluginHost } = createWith({
      settingsHolder: { current: { rest } as never },
      getCommandDeclarations: () => [
        { pluginId: 'md.numen.tree', id: 'tree.open', title: 'Open tree' },
        { pluginId: 'md.numen.cards', id: 'deck.create', title: 'Create deck' },
      ],
      reportCommandIssue,
    })
    pluginHost.createContext('md.numen.tree')
    pluginHost.createContext('md.numen.cards')

    pluginHost.getPaletteCommands()
    pluginHost.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledExactlyOnceWith(
      'md.numen.tree',
      expect.stringContaining('tree.open'),
    )
  })

  it('lists the handlers of a shared command and runs one in its backend', async () => {
    const rest = { plugins: ref([{ id: 'md.numen.pages', hasFrontend: false, hasBackend: true }]) }
    const runBackendHandler = vi.fn(async () => undefined)
    const { pluginHost } = createWith({
      settingsHolder: { current: { rest } as never },
      handlers: {
        getHandlerDeclarations: () => [
          {
            pluginId: 'md.numen.pages',
            command: 'app:file.createText',
            title: 'Extract text layer',
            hasItems: false,
          },
        ],
        runBackendHandler,
      },
    })
    const target = { path: 'scan.pdf', title: 'scan', isFolder: false, others: [], tab: '' }

    expect(await pluginHost.listCommandItems('app:file.createText', target)).toStrictEqual([
      { id: 'md.numen.pages', title: 'Extract text layer' },
    ])
    await pluginHost.executeCommand('app:file.createText', { ...target, type: '' })

    expect(runBackendHandler).toHaveBeenCalledWith({
      pluginId: 'md.numen.pages',
      command: 'app:file.createText',
      target: expect.objectContaining({ path: 'scan.pdf' }),
      itemId: undefined,
    })
  })

  it('rejects a run awaited through the host with the error of a backend handler, and writes nothing', async () => {
    const rest = { plugins: ref([{ id: 'md.numen.ears', hasFrontend: false, hasBackend: true }]) }
    const writeMessage = vi.fn()
    const onError = vi.fn()
    const handlers = createBackendHandlers({
      getHandlers: () => [
        {
          pluginId: 'md.numen.ears',
          command: 'app:file.createText',
          title: 'Transcribe',
          hasItems: false,
        },
      ],
      list: async () => [],
      run: async () => ({ ok: false, error: 'the recording has no audio' }),
      writeMessage,
      openFile: async () => undefined,
    })
    const { pluginHost } = createWith({
      settingsHolder: { current: { rest } as never },
      handlers,
      reportError: onError,
    })
    const note = { path: 'talks/a.mp3', title: 'a', type: '', isFolder: false, others: [], tab: '' }

    await expect(
      pluginHost.executeCommand('app:file.createText', note, { isAwaited: true }),
    ).rejects.toThrow('the recording has no audio')
    expect(writeMessage).not.toHaveBeenCalled()
    expect(onError).not.toHaveBeenCalled()
  })

  it('lists the items of a handler the core lists them for', async () => {
    const rest = { plugins: ref([{ id: 'md.numen.ocr', hasFrontend: false, hasBackend: true }]) }
    const listBackendHandlerItems = vi.fn(async () => [{ id: 'v1', title: 'Recognition' }])
    const { pluginHost } = createWith({
      settingsHolder: { current: { rest } as never },
      handlers: {
        getHandlerDeclarations: () => [
          {
            pluginId: 'md.numen.ocr',
            command: 'app:link.downloadMedia',
            title: 'Delete',
            hasItems: true,
          },
        ],
        listBackendHandlerItems,
      },
    })
    const target = { path: 'scan.pdf', title: 'scan', isFolder: false, others: [], tab: '' }

    expect(await pluginHost.listCommandItems('app:link.downloadMedia', target)).toStrictEqual([
      { id: 'md.numen.ocr:v1', title: 'Recognition', group: 'md.numen.ocr' },
    ])
    expect(listBackendHandlerItems).toHaveBeenCalledWith(
      'md.numen.ocr',
      'app:link.downloadMedia',
      expect.objectContaining({ path: 'scan.pdf' }),
    )
  })

  it('reads no setting before the window has its settings', () => {
    const { pluginHost } = createWith({})

    expect(pluginHost.createContext('md.numen.a').getSetting('enabled')).toBeUndefined()
  })

  it('calls the handler of a followed setting once per changed value', async () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = { getCommittedSetting: (path: readonly string[]) => settings.value[path[1] ?? ''] }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const handler = vi.fn()
    pluginHost.createContext('md.numen.a').onSettingChange('fontSize', handler)

    settings.value = { fontSize: 14 }
    await nextTick()
    settings.value = { fontSize: 16 }
    await nextTick()

    expect(handler).toHaveBeenCalledExactlyOnceWith(16)
  })

  it("shows a plugin's notices in the window's one list, and clears them with the plugin", () => {
    const notices = createNoticeQueue()
    const { pluginHost } = createWith({ notices })

    pluginHost.createContext('md.numen.a').notices?.show('Saved.', 'warning')
    expect(notices.notices.value.map((one) => one.text)).toStrictEqual(['Saved.'])

    pluginHost.disposePlugin('md.numen.a')
    expect(notices.notices.value).toStrictEqual([])
  })

  it('gives a plugin no notices where the window keeps no list', () => {
    const { pluginHost } = createWith({})

    expect(pluginHost.createContext('md.numen.a').notices).toBeUndefined()
  })
})

describe('the plugins the window loads', () => {
  it('are the ones that declare a frontend, so a backend alone is never fetched', () => {
    const plugins = [
      { id: 'md.numen.agents.server', hasFrontend: false },
      { id: 'md.numen.views.sketch', hasFrontend: true },
    ]

    expect(getFrontendPluginIds(plugins)).toStrictEqual(['md.numen.views.sketch'])
  })

  it('tracks frontend plugins when settings change and wires host capabilities', async () => {
    const windowTabs = createTestTabs()
    const reportError = vi.fn()
    const readNote = vi.fn(async () => ({ body: 'text' }))
    const writeNote = vi.fn(async () => undefined)
    const resolveLink = vi.fn(async () => 'Target.md')
    const taskList = ref([])
    const workspace = {
      getNoteInFocus: () => ({ path: 'a.md', title: 'A' }),
    } as unknown as WorkspaceHost
    const notes = { read: vi.fn() } as never
    const relations = { roles: [] } as never
    const files = { list: vi.fn() } as never
    const agent = { finish: vi.fn() } as unknown as AgentHost
    const buffers = { register: vi.fn() } as never
    const noteBuffer = { open: vi.fn() } as never
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const pluginsRef = ref([{ id: 'md.numen.a', hasFrontend: true }])
    const rest = {
      getCommittedSetting: (path: readonly string[]) =>
        path.length === 2 ? settings.value[path[1] ?? ''] : undefined,
      plugins: pluginsRef,
    }
    const { pluginHost } = createPluginSupport({
      windowTabs,
      tabLinks: {} as never,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      settingsHolder: { current: { rest } as never },
      reportError,
      readNote,
      writeNote,
      resolveLink,
      services: {
        workspace,
        notes,
        relations,
        files,
        agent,
        buffers,
        noteBuffer,
        taskList,
      },
    })

    const ctx = pluginHost.createContext('md.numen.a')
    const icon = { name: 'CustomIcon' }
    ctx.registerTabKind({
      kind: 'sample' as never,
      component: null,
      title: 'Sample',
      icon,
    })
    expect(getNoteIcon('sample')).toBe(icon)
    const tabId = await windowTabs.openTabOfKind('sample', 'a.md')
    expect(windowTabs.getTab(tabId!)).toBeDefined()

    // Failing onSettingChange handler reports to reportError
    ctx.onSettingChange('fontSize', (() => Promise.reject(new Error('bad handler'))) as never)
    settings.value = { fontSize: 18 }
    await nextTick()
    await new Promise((r) => setTimeout(r, 0))
    expect(reportError).toHaveBeenCalledWith('md.numen.a', expect.any(Error))

    // Empty pluginId in disposePlugin does not close tabs
    const closeSpy = vi.spyOn(windowTabs, 'closeTabsOfPlugin')
    pluginHost.disposePlugin('')
    expect(closeSpy).not.toHaveBeenCalled()

    // Services and tasks accessors
    expect(ctx.workspace).toBeDefined()
    expect(ctx.notes).toBeDefined()
    expect(ctx.relations).toBeDefined()
    expect(ctx.files).toBeDefined()
    expect(ctx.agent).toBeDefined()
    expect(ctx.buffers).toBeDefined()
    expect(ctx.tasks).toBeDefined()
    expect(ctx.texts).toBeDefined()

    pluginHost.disposePlugin('md.numen.a')
    expect(getNoteIcon('sample')).not.toBe(icon)

    // watchPluginSettings tracks frontend plugins when pluginsRef updates
    const trackPlugins = vi.fn(() => Promise.resolve())
    const watchErr = vi.fn()
    watchPluginSettings({ rest } as never, { trackPlugins } as never, watchErr)
    pluginsRef.value = [
      { id: 'md.numen.a', hasFrontend: true },
      { id: 'md.numen.b', hasFrontend: false },
    ]
    await nextTick()
    expect(trackPlugins).toHaveBeenCalledWith(['md.numen.a'])
  })
})
