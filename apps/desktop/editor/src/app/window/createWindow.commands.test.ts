/**
 * The commands run through the window: the palette runs a command with the registered handler,
 * what the command objects to is written as a message of the window, and the files a plugin is
 * provided run their commands with the dependencies of the window.
 */
import { defineComponent, ref, shallowRef } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { SLOT, type SlotCommand } from '@numen/plugin-loader'
import { createNoticeQueue } from '@numen/workbench'
import type { Task } from '@/entities/window'
import { ReportedError } from '@numen/result'
import { COMMAND } from '@/shared/model/command'
import { createInvocation, PALETTE_WORDS } from '@/features/command-palette'
import { createPluginNotes } from '@/features/plugin-host'
import { createTestTransports } from '@/testing/coreAnswers'
import {
  listCards,
  createWindowTransport,
  getWindowParts,
  mountWindow,
  mountWindowWithPalette,
  createNameAnswer,
  requests,
  settle,
  submitPalette,
  enterPaletteText,
  vaultState,
} from '@/testing/window'
import { createPluginDeclarations } from './createPluginDeclarations'
import { createServiceParts } from './createPluginHost'
import {
  createPluginAccess,
  createPluginBackgroundTasks,
  createPluginWorkspace,
  createWindowOwnedServices,
} from './createPluginServices'
import { createVaultServices } from './createVaultServices'
import { createWindowApi } from './createWindowApi'
import { createWindowBase } from './createWindowBase'
import { createWindowPalettes } from './createWindowPalettes'
import { createWindowPlugins } from './createWindowPlugins'
import { createWindowStart } from './createWindowStart'
import { createWindowTabMenu } from './createWindowTabMenu'
import { registerSettingsPage } from './registerSettingsPage'

describe('a command run through the palette of the window', () => {
  it('is carried out by the handler registered for it', async () => {
    const window = await mountWindowWithPalette()
    const { palette } = getWindowParts(window)

    palette.runCommand(COMMAND.createNote, palette.getTarget())
    await settle()
    await enterPaletteText('Entropy')
    await submitPalette()

    expect(requests.made).toStrictEqual(['Entropy'])
  })

  it('is written as a message of the window where the command is not offered over the target', async () => {
    const window = await mountWindow()
    const { palette } = getWindowParts(window)

    palette.runCommand(COMMAND.openFile, { ...palette.getTarget(), path: '' })
    await settle()

    expect(listCards().join(' ')).toContain(PALETTE_WORDS.noNote)
  })
})

describe('the files a plugin is provided', () => {
  it('run their commands with the dependencies of the window', async () => {
    const window = await mountWindow()
    const { pluginHost } = getWindowParts(window)

    await pluginHost.contextHost.vault.getFiles()?.createFolder('Zoology')

    expect(requests.folders).toStrictEqual(['Zoology'])
  })
})

const hasUnavailableServices = (
  value: object,
): value is { services: { openPlugin: (id?: string) => void } } => 'services' in value

const hasSettingsPlatform = (
  value: object,
): value is { installation: { settings: { getPlatform: () => string } } } => 'installation' in value

function rethrow(error: unknown): never {
  throw error
}

describe('the parts of the window wired together', () => {
  it('wires foundation, base, tabs, vault, declarations, plugins, palettes and start', async () => {
    const baseApi = createWindowApi(
      createTestTransports(
        createWindowTransport({
          connectionCalls: { listRows: () => ({ rows: [] }) },
        }),
      ),
    )
    const watchTabLinks = vi.fn(async function* () {
      yield* []
    })
    const syncTabPath = vi.fn().mockRejectedValueOnce(new ReportedError('sync-failed'))
    const listDeclaredCommands = vi.fn(async () => ({
      commands: [{ pluginId: 'p1', id: 'cmd1', title: 'Cmd 1', arguments: [] }],
      slots: [],
      handlers: [
        { pluginId: 'p1', command: 'cards:review', title: 'Review cards', hasItems: false },
      ],
      keybindings: [{ pluginId: 'p1', command: 'p1:cmd1', key: 'Mod+Shift+K' }],
    }))
    const runDeclaredHandler = vi.fn(async () => ({
      ok: false as const,
      error: 'delete-version-failed',
    }))
    const updateSettings = vi.fn().mockRejectedValueOnce(new ReportedError('cfg-failed'))
    const listConnections = vi.fn().mockRejectedValueOnce(new ReportedError('conn-failed'))
    const cancelTask = vi.fn().mockRejectedValueOnce(new ReportedError('cancel-failed'))
    const api: typeof baseApi = {
      ...baseApi,
      tabs: {
        ...baseApi.tabs,
        tabLinks: {
          ...baseApi.tabs.tabLinks,
          watchTabLinks,
          syncTabPath,
        },
      },
      plugins: {
        ...baseApi.plugins,
        declaredCommands: {
          ...baseApi.plugins.declaredCommands,
          listDeclaredCommands,
          runDeclaredHandler,
        },
      },
      configuration: {
        ...baseApi.configuration,
        settings: {
          ...baseApi.configuration.settings,
          updateSettings,
        },
        connection: {
          ...baseApi.configuration.connection,
          listConnections,
        },
      },
      window: {
        ...baseApi.window,
        tasks: {
          ...baseApi.window.tasks,
          cancelTask,
        },
      },
    }

    expect(api.plugins.pluginHost.getPluginTexts('p1')).toBeDefined()
    expect(
      await api.plugins.pluginHost
        .getPluginConnections('p1', () => ({ dispose: () => undefined }))
        .listRows?.('llm.complete'),
    ).toStrictEqual([])

    const base = createWindowBase(api)
    await base.foundation.rest.write.writeSetting(['app', 'mode'], 'dark').catch(() => undefined)
    await base.foundation.connections.load()
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'configured', severity: 'error' }),
        expect.objectContaining({ name: 'connections', severity: 'error' }),
      ]),
    )

    const onSettingsRequest = vi.fn()
    base.foundation.settingsRequests.subscribe(onSettingsRequest)
    base.tabs.windowTabs.restored.openSavedTab({
      id: 'unavail-1',
      kind: 'unregistered.kind',
      path: 'notes/a.md',
    })
    const unavailTab = base.tabs.windowTabs.tabs.getTab('unavail-1')
    if (unavailTab && typeof unavailTab.state === 'object' && unavailTab.state !== null) {
      if (hasUnavailableServices(unavailTab.state)) {
        unavailTab.state.services.openPlugin('md.example.plugin')
      }
    }
    expect(onSettingsRequest).toHaveBeenCalledWith({ pluginId: 'md.example.plugin' })

    const measureSpy = vi.fn()
    const StubPane = defineComponent({ template: '<div />' })
    base.tabs.windowTabs.kinds.registerKind({
      kind: 'measurable',
      open: () => ({ measure: measureSpy }),
      pane: StubPane,
      getTitle: () => 'Measurable',
      display: { placement: { pane: 'main', order: 1 } },
    })
    base.tabParts.openCurrentVault()
    expect(base.tabs.windowTabs.tabs.tabs.value.some((t) => t.title === 'Measurable')).toBe(true)

    base.windowSettings.appearance.state.settings.value = { interfaceScale: 1.2, textScale: 1.1 }
    await settle()
    expect(measureSpy).toHaveBeenCalled()

    base.tabs.linkSession.start().catch(rethrow)
    await vi.waitFor(() =>
      expect(watchTabLinks).toHaveBeenCalledWith(base.tabs.windowId, expect.any(AbortSignal)),
    )

    base.tabs.tabLinks.applySnapshot({
      groups: [{ id: 'g1', tabs: ['t1', 't2'], path: 'notes/a.md', title: 'A', colorIndex: 0 }],
      tabs: [],
      windows: [],
    })
    base.tabs.tabSync.syncPathFromTab('t1', 'notes/b.md', 'B')
    await settle()
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'command', severity: 'error', text: 'sync-failed' }),
      ]),
    )

    const getWriterSpy = vi.spyOn(base.foundation.log, 'getWriter')
    const vaultCheck = createVaultServices({
      api,
      log: base.foundation.log,
      windowTabs: base.tabs.windowTabs,
      tabOpeners: base.tabs.tabOpeners,
      navigation: base.tabs.navigation,
    })
    expect(getWriterSpy).toHaveBeenCalledWith('command')
    vaultCheck.editing.notes.open('note-tab-1', 'notes/Entropy.md')
    expect(vaultCheck.editing.reached.getPath('note-tab-1')).toBe('notes/Entropy.md')
    expect(await vaultCheck.itemLists.readFileKinds(['notes/Entropy.md'])).toBeInstanceOf(Map)

    base.foundation.rest.state.app.value = [
      {
        key: 'appearance.mode',
        type: 'enum',
        title: 'Mode',
        description: '',
        default: 'dark',
        value: 'dark',
      },
    ]
    base.foundation.rest.state.plugins.value = [
      {
        id: 'p1',
        name: 'Plugin One',
        version: '1.0.0',
        description: '',
        isActive: true,
        capabilities: [],
        hasFrontend: true,
        hasBackend: true,
        settings: [],
      },
    ]

    const openFileSpy = vi.fn(async () => undefined)
    const decl = createPluginDeclarations({
      api: {
        ...api.plugins.declaredCommands,
        runDeclaredHandler: async () => ({
          ok: true as const,
          value: { message: 'handler-ok', openedPath: 'notes/opened.md' },
        }),
      },
      rest: base.foundation.rest,
      log: base.foundation.log,
      isPaletteOpen: () => false,
      getTarget: base.getTarget,
      appCommands: base.appCommands,
      itemLists: {
        ...base.vault.itemLists,
        readFileKinds: vi.fn().mockRejectedValueOnce(new ReportedError('note-kind-failed')),
      },
      tabOpeners: { openFile: openFileSpy },
    })
    expect(decl.declared.whenContext.getConfig()).toStrictEqual({
      app: { appearance: { mode: 'dark' } },
    })
    decl.declared.whenContext.getNote?.({ ...base.getTarget(), path: 'notes/a.md' })
    await settle()
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'command', severity: 'error', text: 'note-kind-failed' }),
      ]),
    )
    await decl.declared.load()
    expect(decl.hostParts[1]?.commands?.getHandlerDeclarations?.()).toStrictEqual([
      { pluginId: 'p1', command: 'cards:review', title: 'Review cards', hasItems: false },
    ])
    expect(decl.hostParts[2]?.commands?.getCommandDeclarations?.()).toStrictEqual([
      { pluginId: 'p1', id: 'cmd1', title: 'Cmd 1' },
    ])
    await decl.hostParts[1]?.commands?.runBackendHandler?.({
      pluginId: 'p1',
      command: 'cards:review',
      target: { path: 'notes/a.md', title: 'A', isFolder: false, others: [], tab: '' },
    })
    expect(openFileSpy).toHaveBeenCalledWith('notes/opened.md', '', 'current')
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'command', severity: 'info', text: 'handler-ok' }),
      ]),
    )

    const plugins = createWindowPlugins({
      foundation: base.foundation,
      tabs: base.tabs,
      vault: base.vault,
      windowSettings: base.windowSettings,
      getTarget: base.getTarget,
      appCommands: base.appCommands,
    })
    const flushDisp = plugins.pluginHost.contextHost.vault
      .getBuffers()
      ?.addFlushHandler(async () => undefined)
    expect(flushDisp?.dispose).toBeTypeOf('function')
    flushDisp?.dispose()
    await plugins.pluginHost.contextHost.vault.getNotes()?.read('notes/Entropy.md')

    const notices = createNoticeQueue()
    const clearSpy = vi.spyOn(notices, 'clear')
    const workspace = createPluginWorkspace({
      log: base.foundation.log,
      tabs: base.tabs,
      window: base.vault.window,
      hostPort: plugins.hostPort,
    })
    expect(workspace.window.getToday()).toMatch(/^\d{4}-\d{2}-\d{2}$/)
    getWriterSpy.mockClear()
    const bgTasks = createPluginBackgroundTasks({
      cancelTask,
      log: base.foundation.log,
      notices,
      window: {
        ...base.vault.window,
        tasks: shallowRef<readonly Task[]>([
          {
            id: 'task-1',
            label: 'Task 1',
            about: '',
            progress: { completedCount: 0, total: 0, counting: 'items' },
            error: '',
            source: { isUserStarted: false, pluginId: '', pluginName: '' },
            isCancellable: true,
            kind: 'work',
          },
        ]),
      },
      appearance: base.windowSettings.appearance,
      commands: plugins.hostPort.commands,
      getCommandTitle: (id) => id,
      getPluginName: (id) => id,
    })
    expect(getWriterSpy).toHaveBeenCalledWith('command')
    await expect(bgTasks.cancel('task-1')).rejects.toThrow('cancel-failed')
    await settle()
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          name: 'command',
          severity: 'error',
          text: 'The task was not cancelled: cancel-failed',
        }),
      ]),
    )

    getWriterSpy.mockClear()
    const owned = createWindowOwnedServices({
      editing: base.vault.editing,
      window: base.vault.window,
      vaults: base.vault.vaults,
      hostPort: plugins.hostPort,
      backgroundTasks: bgTasks,
      frame: {
        ...base.foundation.frame,
        minimizeWindow: vi.fn().mockRejectedValueOnce(new ReportedError('minimize-failed')),
      },
      log: base.foundation.log,
    })
    expect(getWriterSpy).toHaveBeenCalledWith('command')
    owned.window.minimize()
    await settle()
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: 'command', severity: 'error', text: 'minimize-failed' }),
      ]),
    )

    const pluginNotes = createPluginNotes({
      read: api.vault.note.readNote,
      write: api.vault.note.writeNote,
    })
    const serviceParts = createServiceParts({
      api,
      rest: base.foundation.rest,
      services: {
        workspace,
        ...createPluginAccess({
          api: api.vault,
          making: base.vault.editing.making,
          pluginNotes,
          getTarget: base.getTarget,
          hostPort: plugins.hostPort,
        }),
        ...owned,
      },
      notices,
      pluginNotes,
      hostParts: decl.hostParts,
    })
    serviceParts[2]?.window?.clearNotices?.('p1')
    expect(clearSpy).toHaveBeenCalledWith('p1')

    await plugins.declaredCommands.load()
    await settle()
    expect(plugins.pluginHost.keybindings.getKeyFor('p1:cmd1')).toBe('Mod+Shift+K')

    getWriterSpy.mockClear()
    const palettes = createWindowPalettes({
      foundation: base.foundation,
      tabs: base.tabs,
      vault: base.vault,
      windowSettings: base.windowSettings,
      getTarget: base.getTarget,
      appCommands: base.appCommands,
      plugins,
    })
    expect(getWriterSpy).toHaveBeenCalledWith('command')

    await base.foundation.registry.get(COMMAND.deleteText)?.(
      {
        ...createInvocation(COMMAND.deleteText, { ...base.getTarget(), path: 'notes/a.md' }),
        name: 'v1',
      },
      palettes.commandDeps,
    )
    expect(base.foundation.log.messages.value).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          name: 'command',
          severity: 'error',
          text: 'delete-version-failed',
        }),
      ]),
    )

    palettes.commandDeps.navigator.openSettings({ pluginId: 'p1' })
    expect(onSettingsRequest).toHaveBeenCalledWith({ pluginId: 'p1' })
    palettes.commandDeps.navigator.search()
    expect(palettes.palette.search.open.value).toBe(true)

    const renameSlotEntry: SlotCommand = {
      kind: 'command',
      key: 'app/command/rename',
      pluginId: 'app',
      command: COMMAND.renameNote,
      title: 'Rename',
      shortcut: '',
      hasItems: false,
    }
    vi.spyOn(plugins.pluginHost.slots, 'getEntries').mockImplementation((slot) =>
      slot === SLOT.paletteSearchResult ? [renameSlotEntry] : [],
    )
    vaultState.names = [createNameAnswer('notes/Entropy.md', 'Entropy')]
    await palettes.palette.search.setTyped('Entropy')
    await vi.waitFor(() =>
      expect(
        palettes.palette.search.groups.value[0]?.items[0]?.actions?.length ?? 0,
      ).toBeGreaterThan(1),
    )

    getWriterSpy.mockClear()
    const started = createWindowStart({
      foundation: base.foundation,
      tabs: base.tabs,
      tabParts: base.tabParts,
      vault: base.vault,
      windowSettings: base.windowSettings,
      getTarget: base.getTarget,
      plugins,
      palettes,
    })
    expect(getWriterSpy).toHaveBeenCalledWith('command')
    expect(started.views.pluginHost).toBe(plugins.pluginHost)

    const closeSlotEntry: SlotCommand = {
      kind: 'command',
      key: 'app/command/close',
      pluginId: 'app',
      command: COMMAND.closeTab,
      title: 'Close',
      shortcut: '',
      hasItems: false,
    }
    const tabMenuWriter = vi.fn()
    const customTabMenu = createWindowTabMenu({
      windowTabs: base.tabs.windowTabs,
      tabLinks: base.tabs.tabLinks,
      moves: base.tabParts.moves,
      pluginHost: {
        slots: {
          ...plugins.pluginHost.slots,
          getEntries: () => [closeSlotEntry],
        },
      },
      windowCommands: {
        ...palettes.windowCommands,
        run: vi.fn().mockRejectedValueOnce(new ReportedError('tab-menu-failed')),
      },
      itemLists: base.vault.itemLists,
      writeMessage: tabMenuWriter,
    })
    customTabMenu.open('unavail-1', { x: 10, y: 20 })
    const firstMenuItem = customTabMenu.menu.value?.items[0]
    if (firstMenuItem) {
      customTabMenu.choose('unavail-1', firstMenuItem.id)
      await settle()
    }
    expect(tabMenuWriter).toHaveBeenCalledWith('tab-menu-failed', 'error')

    const customTabs = createWindowBase(api).tabs.windowTabs
    registerSettingsPage({
      foundation: {
        ...base.foundation,
        frame: { ...base.foundation.frame, platform: ref('darwin' as const) },
      },
      windowSettings: base.windowSettings,
      windowTabs: customTabs,
      plugins: {
        host: plugins.pluginHost,
        health: base.vault.window.pluginHealth,
        commands: plugins.hostPort.commands,
      },
    })
    const settingsTabId = await customTabs.tabs.openTabOfKind('settings', '')
    const settingsTab = customTabs.tabs.getTab(settingsTabId)
    const platformValue =
      settingsTab &&
      typeof settingsTab.state === 'object' &&
      settingsTab.state !== null &&
      hasSettingsPlatform(settingsTab.state)
        ? settingsTab.state.installation.settings.getPlatform()
        : ''
    expect(platformValue).toBe('darwin')
  })
})
