import { describe, expect, it, vi } from 'vitest'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { nextTick, ref } from 'vue'
import {
  DEFAULT_PLUGIN_ENABLED,
  type PluginContext,
  type WorkspaceHost,
} from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import { createTestTabs } from '@/testing/tabs'
import { blank } from '@/testing/stubs'
import type { AnyTabKind } from '@/entities/tab'
import type { Manifest } from '@numen/core-client'
import { createMockManifest } from '@numen/core-client/testing'
import { createTestPluginSupport, type TestPluginSupportDeps } from '@/testing/pluginSupport'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { getNoteIcon } from '@/entities/note'

const createSupport = () => {
  const windowTabs = createTestTabs()
  const support = createTestPluginSupport({
    windowTabs,
    tabOpeners: {
      registerExtensionReader: vi.fn(() => () => undefined),
      registerEditor: vi.fn(() => () => undefined),
    } as never,
    rest: createSettingsStore(createAnsweredApi({}).settings, SETTINGS_WORDS, vi.fn()),
    getServices: () => ({
      workspace: { getNoteInFocus: () => ({ path: '', title: '' }) } as unknown as WorkspaceHost,
    }),
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

    expect(support.pluginHost.lifecycle.createContext('md.numen.a').plugin.app.platform).toBe(
      'windows',
    )
  })
})

describe('closing the tabs of a plugin', () => {
  it('replaces the tabs of the kinds the plugin registered with placeholders', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost.lifecycle
      .createContext(
        'md.numen.a',
        createMockManifest('md.numen.a', {
          name: 'md.numen.a',
          permissions: ['vault:read'],
          tabKinds: ['a-kind'],
        }),
      )
      .register.tabKind({ kind: 'a-kind', component: blank, title: 'A' })
    const id = await windowTabs.tabs.createTab('a-kind', 'x')
    windowTabs.panes.show(id)

    support.pluginHost.lifecycle.disposePlugin('md.numen.a')

    expect(windowTabs.tabs.getTab(id)?.state).toMatchObject({
      kind: 'a-kind',
      targetPath: 'x',
      saved: undefined,
      title: 'x',
    })
  })

  it('keeps a tab of a core kind whose state names the plugin', async () => {
    const { windowTabs, support } = createSupport()
    windowTabs.groups.registerKind(coreKind({ pluginId: 'md.numen.a' }))
    const id = await windowTabs.tabs.createTab('core', 'x')

    support.pluginHost.lifecycle.disposePlugin('md.numen.a')

    expect(windowTabs.tabs.getTab(id)).not.toBeNull()
  })

  it('keeps the tabs of another plugin', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost.lifecycle
      .createContext(
        'md.numen.b',
        createMockManifest('md.numen.b', {
          name: 'md.numen.b',
          permissions: ['vault:read'],
          tabKinds: ['b-kind'],
        }),
      )
      .register.tabKind({ kind: 'b-kind', component: blank, title: 'B' })
    const id = await windowTabs.tabs.createTab('b-kind', 'x')

    support.pluginHost.lifecycle.disposePlugin('md.numen.a')

    expect(windowTabs.tabs.getTab(id)).not.toBeNull()
  })
})

/** A plugin that adds one tab kind to the window. */
const manifest: Manifest = createMockManifest('md.numen.sample.chat', {
  name: 'Chat',
  permissions: ['vault:read'],
  tabKinds: ['chat'],
})

const activate = (context: PluginContext) => {
  context.register.tabKind({ kind: 'chat', component: blank, title: 'Chat' })
}

describe('the tab kind a plugin adds', () => {
  const activateChat = async () => {
    const { windowTabs, support } = createSupport()
    activate(support.pluginHost.lifecycle.createContext(manifest.id, manifest))
    return { windowTabs, support }
  }

  it('is a kind of the window while the plugin is active', async () => {
    const { windowTabs } = await activateChat()

    expect(windowTabs.groups.hasKind('chat')).toBe(true)
    expect(windowTabs.tabs.getTab(await windowTabs.tabs.createTab('chat', ''))).not.toBeNull()
  })

  it('goes, with its open tabs turned into placeholders, when the plugin is disabled', async () => {
    const { windowTabs, support } = await activateChat()
    const id = await windowTabs.tabs.createTab('chat', '')

    support.pluginHost.lifecycle.disposePlugin(manifest.id)

    expect(windowTabs.groups.hasKind('chat')).toBe(false)
    expect(windowTabs.tabs.getTab(id)?.state).toMatchObject({
      kind: 'chat',
      targetPath: '',
      saved: undefined,
      title: 'Chat',
    })
  })
})

describe('the settings and notices a plugin is given', () => {
  const createWith = (more: Partial<TestPluginSupportDeps>) =>
    createTestPluginSupport({
      windowTabs: createTestTabs(),
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: createSettingsStore(createAnsweredApi({}).settings, SETTINGS_WORDS, vi.fn()),
      ...more,
    })

  it('reads a setting of the plugin, and the default of its switch where none is written', () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = {
      read: { getCommittedSetting: (path: readonly string[]) => settings.value[path[1] ?? ''] },
    }
    const { pluginHost } = createWith({ rest: rest as never })
    const ctx = pluginHost.lifecycle.createContext('md.numen.a')

    expect(ctx.settings.get('fontSize')).toBe(14)
    expect(ctx.settings.get('enabled')).toBe(DEFAULT_PLUGIN_ENABLED)
    expect(ctx.settings.get('missing')).toBeUndefined()
  })

  it('prefers the committed setting over an unconfirmed pending write', () => {
    const rest = {
      read: {
        getCommittedSetting: (path: readonly string[]) =>
          path[1] === 'enabled' ? false : undefined,
        getSetting: () => true,
      },
    }
    const { pluginHost } = createWith({ rest: rest as never })
    const ctx = pluginHost.lifecycle.createContext('md.numen.a')

    expect(ctx.settings.get('enabled')).toBe(false)
  })

  it('registers the extensions declared for a reader in the plugin settings', () => {
    const registerExtensionReader = vi.fn(() => () => undefined)
    const rest = {
      state: {
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
      },
    }
    const { pluginHost } = createWith({
      tabOpeners: {
        registerExtensionReader,
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: rest as never,
    })

    pluginHost.lifecycle
      .createContext('md.numen.pdf', createMockManifest('md.numen.pdf', { readers: ['document'] }))
      .register.reader({ id: 'document', title: 'Document', component: blank })

    expect(registerExtensionReader).toHaveBeenCalledTimes(1)
    expect(registerExtensionReader).toHaveBeenCalledWith('.pdf', expect.any(Function), undefined)
  })

  it('lists the declared commands of a plugin with a frontend while it loads, and none of a plugin without one', () => {
    const rest = {
      state: {
        plugins: ref([
          { id: 'md.numen.tree', hasFrontend: true },
          { id: 'md.numen.job', hasFrontend: false },
        ]),
      },
    }
    const { pluginHost } = createWith({
      rest: rest as never,
      getCommandDeclarations: () => [
        { pluginId: 'md.numen.tree', id: 'tree.open', title: 'Open tree' },
        { pluginId: 'md.numen.job', id: 'job.run', title: 'Run job' },
      ],
    })

    expect(pluginHost.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.numen.tree:tree.open',
    ])
  })

  it('reports a declared command with no handler for a plugin without a backend, and for none with one', () => {
    const rest = {
      state: {
        plugins: ref([
          { id: 'md.numen.tree', hasFrontend: true, hasBackend: false },
          { id: 'md.numen.cards', hasFrontend: true, hasBackend: true },
        ]),
      },
    }
    const reportCommandIssue = vi.fn()
    const { pluginHost } = createWith({
      rest: rest as never,
      getCommandDeclarations: () => [
        { pluginId: 'md.numen.tree', id: 'tree.open', title: 'Open tree' },
        { pluginId: 'md.numen.cards', id: 'deck.create', title: 'Create deck' },
      ],
      reportCommandIssue,
    })
    pluginHost.lifecycle.createContext('md.numen.tree')
    pluginHost.lifecycle.createContext('md.numen.cards')

    pluginHost.commands.getPaletteCommands()
    pluginHost.commands.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledExactlyOnceWith(
      'md.numen.tree',
      expect.stringContaining('tree.open'),
    )
  })

  it('lists the handlers of a shared command and runs one in its backend', async () => {
    const rest = {
      state: { plugins: ref([{ id: 'md.numen.pages', hasFrontend: false, hasBackend: true }]) },
    }
    const runBackendHandler = vi.fn(async () => undefined)
    const { pluginHost } = createWith({
      rest: rest as never,
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

    expect(await pluginHost.commands.listCommandItems('app:file.createText', target)).toStrictEqual(
      [{ id: 'md.numen.pages', title: 'Extract text layer' }],
    )
    await pluginHost.commands.executeCommand('app:file.createText', { ...target, type: '' })

    expect(runBackendHandler).toHaveBeenCalledWith({
      pluginId: 'md.numen.pages',
      command: 'app:file.createText',
      target: expect.objectContaining({ path: 'scan.pdf' }),
      itemId: undefined,
    })
  })

  it('rejects a run awaited through the host with the error of a backend handler', async () => {
    const rest = {
      state: { plugins: ref([{ id: 'md.numen.ears', hasFrontend: false, hasBackend: true }]) },
    }
    const onError = vi.fn()
    const handlers = {
      getHandlerDeclarations: () => [
        {
          pluginId: 'md.numen.ears',
          command: 'app:file.createText',
          title: 'Transcribe',
          hasItems: false,
        },
      ],
      listBackendHandlerItems: async () => [],
      runBackendHandler: async () => {
        throw new Error('the recording has no audio')
      },
    }
    const { pluginHost } = createWith({
      rest: rest as never,
      handlers,
      reportError: onError,
    })
    const note = { path: 'talks/a.mp3', title: 'a', type: '', isFolder: false, others: [], tab: '' }

    await expect(
      pluginHost.commands.executeCommand('app:file.createText', note, { isAwaited: true }),
    ).rejects.toThrow('the recording has no audio')
    expect(onError).not.toHaveBeenCalled()
  })

  it('lists the items of a handler the core lists them for', async () => {
    const rest = {
      state: { plugins: ref([{ id: 'md.numen.ocr', hasFrontend: false, hasBackend: true }]) },
    }
    const listBackendHandlerItems = vi.fn(async () => [{ id: 'v1', title: 'Recognition' }])
    const { pluginHost } = createWith({
      rest: rest as never,
      handlers: {
        getHandlerDeclarations: () => [
          {
            pluginId: 'md.numen.ocr',
            command: 'app:file.createText',
            title: 'Delete',
            hasItems: true,
          },
        ],
        listBackendHandlerItems,
      },
    })
    const target = { path: 'scan.pdf', title: 'scan', isFolder: false, others: [], tab: '' }

    expect(await pluginHost.commands.listCommandItems('app:file.createText', target)).toStrictEqual(
      [{ id: 'md.numen.ocr:v1', title: 'Recognition', group: 'md.numen.ocr' }],
    )
    expect(listBackendHandlerItems).toHaveBeenCalledWith(
      'md.numen.ocr',
      'app:file.createText',
      expect.objectContaining({ path: 'scan.pdf' }),
    )
  })

  it('calls the handler of a followed setting once per changed value', async () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = {
      read: { getCommittedSetting: (path: readonly string[]) => settings.value[path[1] ?? ''] },
    }
    const { pluginHost } = createWith({ rest: rest as never })
    const handler = vi.fn()
    pluginHost.lifecycle.createContext('md.numen.a').settings.onChange('fontSize', handler)

    settings.value = { fontSize: 14 }
    await nextTick()
    settings.value = { fontSize: 16 }
    await nextTick()

    expect(handler).toHaveBeenCalledExactlyOnceWith(16)
  })

  it("shows a plugin's notices in the window's one list, and clears them with the plugin", () => {
    const notices = createNoticeQueue()
    const { pluginHost } = createWith({ notices })

    pluginHost.lifecycle.createContext('md.numen.a').ui.notices.show('Saved.', 'warning')
    expect(notices.notices.value.map((one) => one.text)).toStrictEqual(['Saved.'])

    pluginHost.lifecycle.disposePlugin('md.numen.a')
    expect(notices.notices.value).toStrictEqual([])
  })

  it('drops what a plugin reports where the window keeps no list', () => {
    const { pluginHost } = createWith({})

    expect(() =>
      pluginHost.lifecycle.createContext('md.numen.a').ui.notices.show('Saved.'),
    ).not.toThrow()
  })
})

describe('the plugins the window loads', () => {
  it('wires host capabilities', async () => {
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
    const buffers = { register: vi.fn() } as never
    const noteBuffer = { open: vi.fn() } as never
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const pluginsRef = ref([{ id: 'md.numen.a', hasFrontend: true }])
    const rest = {
      read: {
        getCommittedSetting: (path: readonly string[]) =>
          path.length === 2 ? settings.value[path[1] ?? ''] : undefined,
      },
      state: { plugins: pluginsRef },
    }
    const { pluginHost } = createTestPluginSupport({
      windowTabs,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      rest: rest as never,
      reportError,
      readNote,
      writeNote,
      resolveLink,
      getServices: () => ({
        workspace,
        notes,
        relations,
        files,
        buffers,
        noteBuffer,
        taskList,
      }),
    })

    const ctx = pluginHost.lifecycle.createContext(
      'md.numen.a',
      createMockManifest('md.numen.a', {
        name: 'A',
        permissions: ['vault:read'],
        tabKinds: ['sample'],
      }),
    )
    const icon = { name: 'CustomIcon' }
    ctx.register.tabKind({
      kind: 'sample' as never,
      component: blank,
      title: 'Sample',
      icon,
    })
    expect(getNoteIcon('sample')).toBe(icon)
    const tabId = await windowTabs.tabs.openTabOfKind('sample', 'a.md')
    expect(windowTabs.tabs.getTab(tabId!)).toBeDefined()

    // Failing onSettingChange handler reports to reportError
    ctx.settings.onChange('fontSize', (() => Promise.reject(new Error('bad handler'))) as never)
    settings.value = { fontSize: 18 }
    await nextTick()
    await new Promise((r) => setTimeout(r, 0))
    expect(reportError).toHaveBeenCalledWith('md.numen.a', expect.any(Error))

    // Empty pluginId in disposePlugin does not close tabs
    const closeSpy = vi.spyOn(windowTabs.groups, 'closeTabsOfPlugin')
    pluginHost.lifecycle.disposePlugin('')
    expect(closeSpy).not.toHaveBeenCalled()

    // Services and tasks accessors
    expect(ctx.ui.workspace).toBeDefined()
    expect(ctx.vault.notes).toBeDefined()
    expect(ctx.vault.relations).toBeDefined()
    expect(ctx.vault.files).toBeDefined()
    expect(ctx.vault.buffers).toBeDefined()
    expect(ctx.work.tasks).toBeDefined()
    expect(ctx.vault.texts).toBeDefined()

    pluginHost.lifecycle.disposePlugin('md.numen.a')
    expect(getNoteIcon('sample')).not.toBe(icon)
  })
})
