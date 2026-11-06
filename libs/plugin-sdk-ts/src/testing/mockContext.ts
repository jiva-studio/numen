import type {
  AgentProviderRegistration,
  AppInfo,
  CommandNote,
  CommandRegistration,
  ContextMenuRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  SettingsViewRegistration,
  SlotViewRegistration,
  StatusViewRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '../types'
import { asFailure, asValue, type Result } from '../result'
import type { MountedView, ViewRegistration } from '../views'
import type { DropRegistration } from '../drops'
import {
  createMockBackgroundTaskHost,
  createMockCommandsHost,
  createMockSlotHost,
  createMockVaultsHost,
  createMockWindowHost,
  type MockBackgroundTaskHost,
  type MockCommandsHost,
  type MockSlotHost,
  type MockVaultsHost,
  type MockWindowHost,
} from './mockWindowHosts'

// MockMountedView records one mountView call of a plugin under test, and whether its view was disposed.
export interface MockMountedView {
  readonly capability: string
  readonly element: HTMLElement
  readonly context: unknown
  isDisposed: boolean
  focusCount: number
}

// MockPluginContext extends PluginContext with inspection and simulation methods.
export interface MockPluginContext extends PluginContext {
  readonly commands: MockCommandsHost
  readonly slots: MockSlotHost
  readonly vaults: MockVaultsHost
  readonly window: MockWindowHost
  readonly backgroundTasks: MockBackgroundTaskHost
  getRegisteredReaders(): ReaderRegistration[]
  getRegisteredSettingsViews(): SettingsViewRegistration[]
  getRegisteredStatusViews(): StatusViewRegistration[]
  getRegisteredSlotViews(): SlotViewRegistration[]
  getOpenedAddresses(): string[]
  getRegisteredTabKinds(): TabKindRegistration[]
  getRegisteredCommands(): CommandRegistration[]
  getRegisteredContextMenus(): ContextMenuRegistration[]
  getRegisteredAgentProviders(): AgentProviderRegistration[]
  getRegisteredViews(): ViewRegistration[]
  getRegisteredDropHandlers(): DropRegistration[]
  getMountedViews(): MockMountedView[]
  triggerCommand(commandId: string, note?: CommandNote): Promise<void>
  setSetting<T = unknown>(key: string, value: T): void
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

// Calls the listener and rethrows a rejection as an uncaught error, so a failing handler stays visible.
function notifyListener<T>(listener: (event: T) => void | Promise<void>, event: T): void {
  Promise.resolve(listener(event)).catch((error: unknown) => {
    queueMicrotask(() => {
      throw error
    })
  })
}

const DEFAULT_APP: AppInfo = { name: 'numen', version: '0.0.0-test', build: 1, platform: 'linux' }

const EMPTY_NOTE: CommandNote = { path: '', title: '', type: 'note', isFolder: false, others: [] }

// Reports whether the address is an absolute http or https address, as the window opens only those.
function isWebAddress(url: string): boolean {
  try {
    const { protocol } = new URL(url)
    return protocol === 'http:' || protocol === 'https:'
  } catch {
    return false
  }
}

const OPTIONAL_SERVICE_KEYS = [
  'workspace',
  'notes',
  'relations',
  'files',
  'agent',
  'buffers',
  'texts',
  'notices',
  'closeTab',
] as const

function getOptionalServices(initialValues?: Partial<PluginContext>): Partial<PluginContext> {
  if (!initialValues) return {}
  const picked: Partial<PluginContext> = {}
  for (const key of OPTIONAL_SERVICE_KEYS) {
    if (initialValues[key] !== undefined) {
      Object.assign(picked, { [key]: initialValues[key] })
    }
  }
  return picked
}

// createMockPluginContext builds a controllable context for unit tests.
export function createMockPluginContext(
  initialValues?: Partial<PluginContext> & {
    pluginId?: string
    initialSettings?: Record<string, unknown>
  },
): MockPluginContext {
  const init = initialValues ?? {}
  const readers: ReaderRegistration[] = []
  const settingsViews: SettingsViewRegistration[] = []
  const statusViews: StatusViewRegistration[] = []
  const slotViews: SlotViewRegistration[] = []
  const openedAddresses: string[] = []
  const tabKinds: TabKindRegistration[] = []
  const commands: CommandRegistration[] = []
  const contextMenus: ContextMenuRegistration[] = []
  const agentProviders: AgentProviderRegistration[] = []
  const views: ViewRegistration[] = []
  const drops: DropRegistration[] = []
  const mounted: MockMountedView[] = []
  const settings = new Map<string, unknown>(Object.entries(init.initialSettings ?? {}))
  const settingListeners = new Map<string, Set<(val: unknown) => void>>()
  const vaultListeners = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const fileSelectListeners = new Set<(event: FileSelectEvent) => void | Promise<void>>()
  const subscriptions: Disposable[] = []

  const context: MockPluginContext = {
    pluginId: init.pluginId ?? 'test.plugin',
    target: init.target ?? 'desktop.editor',
    subscriptions,
    app: init.app ?? DEFAULT_APP,
    ...getOptionalServices(init),
    commands: createMockCommandsHost(),
    slots: createMockSlotHost(),
    vaults: createMockVaultsHost(),
    window: createMockWindowHost(),
    backgroundTasks: createMockBackgroundTaskHost(),

    openExternal(url: string): Promise<Result<null>> {
      if (!isWebAddress(url)) return Promise.resolve(asFailure(`not a web address: ${url}`))
      openedAddresses.push(url)
      return Promise.resolve(asValue(null))
    },

    registerReader(registration: ReaderRegistration): Disposable {
      return registerInList(readers, registration, subscriptions)
    },

    registerSettingsView(registration: SettingsViewRegistration): Disposable {
      return registerInList(settingsViews, registration, subscriptions)
    },

    registerStatusView(registration: StatusViewRegistration): Disposable {
      return registerInList(statusViews, registration, subscriptions)
    },

    registerSlotView(registration: SlotViewRegistration): Disposable {
      return registerInList(slotViews, registration, subscriptions)
    },

    registerTabKind<State>(registration: TabKindRegistration<State>): Disposable {
      return registerInList(tabKinds, registration, subscriptions)
    },

    registerCommand(registration: CommandRegistration): Disposable {
      return registerInList(commands, registration, subscriptions)
    },

    registerContextMenu(registration: ContextMenuRegistration): Disposable {
      return registerInList(contextMenus, registration, subscriptions)
    },

    registerAgentProvider(registration: AgentProviderRegistration): Disposable {
      return registerInList(agentProviders, registration, subscriptions)
    },

    registerDropHandler(registration: DropRegistration): Disposable {
      return registerInList(drops, registration, subscriptions)
    },

    registerView(registration: ViewRegistration): Disposable {
      return registerInList(views, registration, subscriptions)
    },

    mountView(capability: string, element: HTMLElement, viewContext: unknown): MountedView {
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
    },

    getSetting<T = unknown>(key: string): T | undefined {
      return settings.get(key) as T | undefined
    },

    onSettingChange<T = unknown>(key: string, handler: (value: T) => void): Disposable {
      let listeners = settingListeners.get(key)
      if (!listeners) {
        listeners = new Set()
        settingListeners.set(key, listeners)
      }
      return addSetListener(listeners, handler as (val: unknown) => void, subscriptions)
    },

    setSetting<T = unknown>(key: string, value: T): void {
      settings.set(key, value)
      const listeners = settingListeners.get(key)
      if (listeners) {
        for (const listener of listeners) {
          listener(value)
        }
      }
    },

    onVaultChange(handler: (event: VaultChangeEvent) => void | Promise<void>): Disposable {
      return addSetListener(vaultListeners, handler, subscriptions)
    },

    onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable {
      return addSetListener(fileSelectListeners, handler, subscriptions)
    },

    emitVaultChange(event: VaultChangeEvent): void {
      for (const listener of vaultListeners) {
        notifyListener(listener, event)
      }
    },

    emitFileSelect(event: FileSelectEvent): void {
      for (const listener of fileSelectListeners) {
        notifyListener(listener, event)
      }
    },

    openTab(kind: string, path?: string): Promise<string> | void {
      if (init.openTab) return init.openTab(kind, path)
      return Promise.resolve(`${kind}:${path ?? ''}`)
    },

    navigateTab(kind: string, path: string): Promise<string> | void {
      if (init.navigateTab) return init.navigateTab(kind, path)
      return Promise.resolve(`${kind}:${path}`)
    },

    getRegisteredReaders(): ReaderRegistration[] {
      return [...readers]
    },

    getRegisteredSettingsViews(): SettingsViewRegistration[] {
      return [...settingsViews]
    },

    getRegisteredStatusViews(): StatusViewRegistration[] {
      return [...statusViews]
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

    getRegisteredContextMenus(): ContextMenuRegistration[] {
      return [...contextMenus]
    },

    getRegisteredAgentProviders(): AgentProviderRegistration[] {
      return [...agentProviders]
    },

    getRegisteredViews(): ViewRegistration[] {
      return [...views]
    },

    getRegisteredDropHandlers(): DropRegistration[] {
      return [...drops]
    },

    getMountedViews(): MockMountedView[] {
      return [...mounted]
    },

    async triggerCommand(commandId: string, note?: CommandNote): Promise<void> {
      const cmd = commands.find((c) => c.id === commandId)
      if (!cmd) {
        throw new Error(`Command not registered: ${commandId}`)
      }
      await cmd.handler(note ?? { ...EMPTY_NOTE })
    },

    dispose(): void {
      while (subscriptions.length > 0) {
        const d = subscriptions.pop()
        d?.dispose()
      }
      vaultListeners.clear()
      fileSelectListeners.clear()
      settingListeners.clear()
    },
  }

  return context
}
