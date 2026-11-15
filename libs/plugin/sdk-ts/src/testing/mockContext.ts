import type {
  AppInfo,
  CommandNote,
  CommandRegistration,
  Disposable,
  HostServices,
  PluginContext,
  PluginEvents,
  PluginRegistrations,
  PluginTabs,
  PluginUi,
  PluginVault,
  PluginWork,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotViewRegistration,
  TabKindRegistration,
  FileSelectEvent,
  JsonValue,
  VaultChangeEvent,
} from '../types'
import type { BufferHost } from '../buffers'
import type { NoticeHost } from '../notices'
import type { VaultTexts } from '../texts'
import type { VaultFiles, VaultNotes, VaultRelations, WorkspaceHost } from '../workspace'
import { asFailure, asValue, type Result } from '../result'
import type { MountedView, ViewCapability, ViewContexts, ViewRegistration } from '../views'
import type { Caught } from '../errors'
import type { DropRegistration } from '../drops'
import { createMockNoticeHost } from './mockNotices'
import {
  createMockBackgroundTaskHost,
  createMockCommandsHost,
  createMockHealthHost,
  createMockProcessHost,
  createMockSlotHost,
  createMockVaultsHost,
  createMockWindowHost,
  type MockBackgroundTaskHost,
  type MockCommandsHost,
  type MockHealthHost,
  type MockProcessHost,
  type MockSlotHost,
  type MockVaultsHost,
  type MockWindowHost,
} from './mockWindowHosts'

// MockMountedView records one mountView call of a plugin under test, and whether its view was disposed.
export interface MockMountedView {
  readonly capability: ViewCapability
  readonly element: HTMLElement
  readonly context: ViewContexts[ViewCapability]
  isDisposed: boolean
  focusCount: number
}

// MockPluginUi is the window part of mock host services, with the hosts a test sets and reads back.
export interface MockPluginUi extends PluginUi {
  readonly commands: MockCommandsHost
  readonly slots: MockSlotHost
  readonly window: MockWindowHost
}

export interface MockPluginVault extends PluginVault {
  readonly vaults: MockVaultsHost
}

export interface MockPluginWork extends PluginWork {
  readonly backgroundTasks: MockBackgroundTaskHost
  readonly processes: MockProcessHost
}

// MockPluginContextInit is what a test gives a mock plugin context: who the plugin is and the
// settings it starts with.
export interface MockPluginContextInit {
  readonly pluginId?: string
  readonly target?: string
  readonly app?: AppInfo
  readonly initialSettings?: Record<string, JsonValue>
}

// MockHostServicesInit is the services a test hands the plugin through mock host services.
export interface MockHostServicesInit {
  readonly workspace?: WorkspaceHost
  readonly notes?: VaultNotes
  readonly relations?: VaultRelations
  readonly files?: VaultFiles
  readonly buffers?: BufferHost
  readonly texts?: VaultTexts
  readonly notices?: NoticeHost
  readonly openTab?: PluginTabs['open']
  readonly navigateTab?: PluginTabs['navigate']
  readonly closeTab?: PluginTabs['close']
}

// MockPluginContext extends PluginContext with inspection and simulation methods.
export interface MockPluginContext extends PluginContext {
  readonly health: MockHealthHost
  getRegisteredReaders(): ReaderRegistration[]
  getRegisteredSettingsViews(): SettingsViewRegistration[]
  getRegisteredSlotViews(): SlotViewRegistration[]
  getOpenedAddresses(): string[]
  getRegisteredTabKinds(): TabKindRegistration[]
  getRegisteredCommands(): CommandRegistration[]
  getRegisteredViews(): ViewRegistration[]
  getRegisteredDropHandlers(): DropRegistration[]
  triggerCommand(
    commandId: string,
    note?: CommandNote,
    itemId?: string,
    values?: Readonly<Record<string, string>>,
  ): Promise<void>
  setSetting(key: string, value: JsonValue): void
  dispose(): void
}

// MockHostServices extends HostServices with the hosts a test sets and reads back.
export interface MockHostServices extends HostServices {
  readonly ui: MockPluginUi
  readonly vault: MockPluginVault
  readonly work: MockPluginWork
  getMountedViews(): MockMountedView[]
  dispose(): void
}

function addDisposable(subscriptions: Disposable[], disposable: Disposable): Disposable {
  subscriptions.push(disposable)
  return disposable
}

function registerInList<T>(list: T[], item: T, subscriptions: Disposable[]): Disposable {
  list.push(item)
  return addDisposable(subscriptions, {
    dispose: () => {
      const idx = list.indexOf(item)
      if (idx !== -1) {
        list.splice(idx, 1)
      }
    },
  })
}

function addSetListener<T>(listeners: Set<T>, handler: T, subscriptions: Disposable[]): Disposable {
  listeners.add(handler)
  return addDisposable(subscriptions, {
    dispose: () => {
      listeners.delete(handler)
    },
  })
}

function notifyListener<T>(listener: (event: T) => void | Promise<void>, event: T): void {
  Promise.resolve(listener(event)).catch((error: Caught) => {
    queueMicrotask(() => {
      throw error
    })
  })
}

const DEFAULT_APP: AppInfo = { name: 'numen', version: '0.0.0-test', build: 1, platform: 'linux' }

const EMPTY_NOTE: CommandNote = {
  path: '',
  title: '',
  type: 'note',
  isFolder: false,
  others: [],
  tab: '',
}

function isWebAddress(url: string): boolean {
  try {
    const { protocol } = new URL(url)
    return protocol === 'http:' || protocol === 'https:'
  } catch {
    return false
  }
}

function pickDefinedParts<T extends object>(parts: { [K in keyof T]: T[K] | undefined }): T {
  const picked: Partial<T> = {}
  for (const key of Object.keys(parts) as (keyof T)[]) {
    const value = parts[key]
    if (value !== undefined) picked[key] = value
  }
  return picked as T
}

// createMockPluginContext builds a controllable plugin context for unit tests.
export function createMockPluginContext(initialValues?: MockPluginContextInit): MockPluginContext {
  const init = initialValues ?? {}
  const readers: ReaderRegistration[] = []
  const settingsViews: SettingsViewRegistration[] = []
  const slotViews: SlotViewRegistration[] = []
  const openedAddresses: string[] = []
  const tabKinds: TabKindRegistration[] = []
  const commands: CommandRegistration[] = []
  const views: ViewRegistration[] = []
  const drops: DropRegistration[] = []
  const settings = new Map<string, JsonValue>(Object.entries(init.initialSettings ?? {}))
  const settingListeners = new Map<string, Set<(val: JsonValue) => void>>()
  const subscriptions: Disposable[] = []

  const register: PluginRegistrations = {
    reader: (registration) => registerInList(readers, registration, subscriptions),
    settingsView: (registration) => registerInList(settingsViews, registration, subscriptions),
    slotView: (registration) => registerInList(slotViews, registration, subscriptions),
    tabKind: <State extends object>(registration: TabKindRegistration<State>) =>
      registerInList(tabKinds, registration, subscriptions),
    command: (registration) => registerInList(commands, registration, subscriptions),
    dropHandler: (registration) => registerInList(drops, registration, subscriptions),
    view: (registration) => registerInList(views, registration, subscriptions),
  }

  const getSettingValue = <T extends JsonValue = JsonValue>(key: string): T | undefined =>
    settings.get(key) as T | undefined

  const onSettingChange = <T extends JsonValue = JsonValue>(
    key: string,
    handler: (value: T) => void,
  ): Disposable => {
    let listeners = settingListeners.get(key)
    if (!listeners) {
      listeners = new Set()
      settingListeners.set(key, listeners)
    }
    return addSetListener(listeners, handler as (val: JsonValue) => void, subscriptions)
  }

  return {
    identity: {
      id: init.pluginId ?? 'test.plugin',
      target: init.target ?? 'desktop.editor',
      subscriptions,
      app: init.app ?? DEFAULT_APP,
      openExternal(url: string): Promise<Result<null>> {
        if (!isWebAddress(url)) return Promise.resolve(asFailure(`not a web address: ${url}`))
        openedAddresses.push(url)
        return Promise.resolve(asValue(null))
      },
    },
    register,
    settings: { get: getSettingValue, onChange: onSettingChange },
    health: createMockHealthHost(),

    setSetting(key: string, value: JsonValue): void {
      settings.set(key, value)
      const listeners = settingListeners.get(key)
      if (listeners) {
        for (const listener of listeners) {
          listener(value)
        }
      }
    },

    getRegisteredReaders(): ReaderRegistration[] {
      return [...readers]
    },

    getRegisteredSettingsViews(): SettingsViewRegistration[] {
      return [...settingsViews]
    },

    getRegisteredSlotViews(): SlotViewRegistration[] {
      return [...slotViews]
    },

    getOpenedAddresses(): string[] {
      return [...openedAddresses]
    },

    getRegisteredTabKinds(): TabKindRegistration[] {
      return [...tabKinds]
    },

    getRegisteredCommands(): CommandRegistration[] {
      return [...commands]
    },

    getRegisteredViews(): ViewRegistration[] {
      return [...views]
    },

    getRegisteredDropHandlers(): DropRegistration[] {
      return [...drops]
    },

    async triggerCommand(
      commandId: string,
      note?: CommandNote,
      itemId?: string,
      values: Readonly<Record<string, string>> = {},
    ): Promise<void> {
      const cmd = commands.find((c) => c.id === commandId)
      if (!cmd) {
        throw new Error(`Command not registered: ${commandId}`)
      }
      await cmd.handler(note ?? { ...EMPTY_NOTE }, itemId, values)
    },

    dispose(): void {
      while (subscriptions.length > 0) {
        const d = subscriptions.pop()
        d?.dispose()
      }
      settingListeners.clear()
    },
  }
}

function createMockMountView(mounted: MockMountedView[]): PluginUi['mountView'] {
  return <C extends ViewCapability>(
    capability: C,
    element: HTMLElement,
    viewContext: ViewContexts[C],
  ): MountedView => {
    const view: MockMountedView = {
      capability,
      element,
      context: viewContext,
      isDisposed: false,
      focusCount: 0,
    }
    mounted.push(view)
    return {
      focus: () => {
        view.focusCount++
      },
      dispose: () => {
        view.isDisposed = true
      },
    }
  }
}

function createMockTabs(init: MockHostServicesInit): PluginTabs {
  return {
    open: (kind, path) => {
      if (init.openTab) return init.openTab(kind, path)
      return Promise.resolve(`${kind}:${path ?? ''}`)
    },
    navigate: (kind, path) => {
      if (init.navigateTab) return init.navigateTab(kind, path)
      return Promise.resolve(`${kind}:${path}`)
    },
    close: init.closeTab ?? (() => undefined),
  }
}

// createMockHostServices builds controllable host services for unit tests, apart from any plugin
// context.
export function createMockHostServices(initialValues?: MockHostServicesInit): MockHostServices {
  const init = initialValues ?? {}
  const mounted: MockMountedView[] = []
  const vaultListeners = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const fileSelectListeners = new Set<(event: FileSelectEvent) => void | Promise<void>>()
  const subscriptions: Disposable[] = []

  const events: PluginEvents = {
    onVaultChange: (handler) => addSetListener(vaultListeners, handler, subscriptions),
    onFileSelect: (handler) => addSetListener(fileSelectListeners, handler, subscriptions),
    emitVaultChange: (event) => {
      for (const listener of vaultListeners) notifyListener(listener, event)
    },
    emitFileSelect: (event) => {
      for (const listener of fileSelectListeners) notifyListener(listener, event)
    },
  }

  return {
    vault: {
      ...pickDefinedParts<Omit<PluginVault, 'vaults'>>({
        notes: init.notes,
        relations: init.relations,
        files: init.files,
        texts: init.texts,
        buffers: init.buffers,
      }),
      vaults: createMockVaultsHost(),
    },
    tabs: createMockTabs(init),
    ui: {
      ...pickDefinedParts<Pick<PluginUi, 'workspace'>>({ workspace: init.workspace }),
      notices: init.notices ?? createMockNoticeHost(),
      commands: createMockCommandsHost(),
      slots: createMockSlotHost(),
      window: createMockWindowHost(),
      mountView: createMockMountView(mounted),
    },
    work: {
      backgroundTasks: createMockBackgroundTaskHost(),
      processes: createMockProcessHost(),
    },
    events,

    getMountedViews(): MockMountedView[] {
      return [...mounted]
    },

    dispose(): void {
      while (subscriptions.length > 0) {
        const d = subscriptions.pop()
        d?.dispose()
      }
      vaultListeners.clear()
      fileSelectListeners.clear()
    },
  }
}
