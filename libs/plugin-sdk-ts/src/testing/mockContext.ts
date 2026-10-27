import type {
  AgentProviderRegistration,
  CommandNote,
  CommandRegistration,
  ContextMenuRegistration,
  Disposable,
  FileSelectEvent,
  PluginContext,
  ReaderRegistration,
  TabKindRegistration,
  VaultChangeEvent,
} from '../types'
import type { MountedView, ViewRegistration } from '../views'
import type { DropRegistration } from '../drops'

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
  getRegisteredReaders(): ReaderRegistration[]
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

// createMockPluginContext builds a controllable context for unit tests.
export function createMockPluginContext(
  initialValues?: Partial<PluginContext> & {
    pluginId?: string
    initialSettings?: Record<string, unknown>
  },
): MockPluginContext {
  const pluginId = initialValues?.pluginId ?? 'test.plugin'
  const readers: ReaderRegistration[] = []
  const tabKinds: TabKindRegistration[] = []
  const commands: CommandRegistration[] = []
  const contextMenus: ContextMenuRegistration[] = []
  const agentProviders: AgentProviderRegistration[] = []
  const views: ViewRegistration[] = []
  const drops: DropRegistration[] = []
  const mounted: MockMountedView[] = []
  const settings = new Map<string, unknown>(Object.entries(initialValues?.initialSettings ?? {}))
  const settingListeners = new Map<string, Set<(val: unknown) => void>>()
  const vaultListeners = new Set<(event: VaultChangeEvent) => void | Promise<void>>()
  const fileSelectListeners = new Set<(event: FileSelectEvent) => void | Promise<void>>()
  const subscriptions: Disposable[] = []

  function addDisposable(d: Disposable): Disposable {
    subscriptions.push(d)
    return d
  }

  const context: MockPluginContext = {
    pluginId,
    target: initialValues?.target ?? 'desktop.editor',
    subscriptions,
    ...(initialValues?.workspace ? { workspace: initialValues.workspace } : {}),
    ...(initialValues?.notes ? { notes: initialValues.notes } : {}),
    ...(initialValues?.files ? { files: initialValues.files } : {}),
    ...(initialValues?.agent ? { agent: initialValues.agent } : {}),
    ...(initialValues?.buffers ? { buffers: initialValues.buffers } : {}),
    ...(initialValues?.texts ? { texts: initialValues.texts } : {}),
    ...(initialValues?.notices ? { notices: initialValues.notices } : {}),
    ...(initialValues?.closeTab ? { closeTab: initialValues.closeTab } : {}),

    registerReader(registration: ReaderRegistration): Disposable {
      readers.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = readers.indexOf(registration)
          if (idx !== -1) {
            readers.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    registerTabKind<State>(registration: TabKindRegistration<State>): Disposable {
      tabKinds.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = tabKinds.indexOf(registration)
          if (idx !== -1) {
            tabKinds.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    registerCommand(registration: CommandRegistration): Disposable {
      commands.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = commands.indexOf(registration)
          if (idx !== -1) {
            commands.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    registerContextMenu(registration: ContextMenuRegistration): Disposable {
      contextMenus.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = contextMenus.indexOf(registration)
          if (idx !== -1) {
            contextMenus.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    registerAgentProvider(registration: AgentProviderRegistration): Disposable {
      agentProviders.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = agentProviders.indexOf(registration)
          if (idx !== -1) {
            agentProviders.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    registerDropHandler(registration: DropRegistration): Disposable {
      drops.push(registration)
      return addDisposable({
        dispose: () => {
          const idx = drops.indexOf(registration)
          if (idx !== -1) drops.splice(idx, 1)
        },
      })
    },

    registerView(registration: ViewRegistration): Disposable {
      views.push(registration)
      const disposable: Disposable = {
        dispose: () => {
          const idx = views.indexOf(registration)
          if (idx !== -1) {
            views.splice(idx, 1)
          }
        },
      }
      return addDisposable(disposable)
    },

    mountView(capability: string, element: HTMLElement, context: unknown): MountedView {
      const view: MockMountedView = {
        capability,
        element,
        context,
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
      const genericHandler = handler as (val: unknown) => void
      listeners.add(genericHandler)
      const disposable: Disposable = {
        dispose: () => {
          listeners?.delete(genericHandler)
        },
      }
      return addDisposable(disposable)
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
      vaultListeners.add(handler)
      const disposable: Disposable = {
        dispose: () => {
          vaultListeners.delete(handler)
        },
      }
      return addDisposable(disposable)
    },

    onFileSelect(handler: (event: FileSelectEvent) => void | Promise<void>): Disposable {
      fileSelectListeners.add(handler)
      const disposable: Disposable = {
        dispose: () => {
          fileSelectListeners.delete(handler)
        },
      }
      return addDisposable(disposable)
    },

    emitVaultChange(event: VaultChangeEvent): void {
      for (const listener of vaultListeners) {
        void listener(event)
      }
    },

    emitFileSelect(event: FileSelectEvent): void {
      for (const listener of fileSelectListeners) {
        void listener(event)
      }
    },

    openTab(kind: string, path?: string): Promise<string> | void {
      if (initialValues?.openTab) return initialValues.openTab(kind, path)
      return Promise.resolve(`${kind}:${path ?? ''}`)
    },

    navigateTab(kind: string, path: string): Promise<string> | void {
      if (initialValues?.navigateTab) return initialValues.navigateTab(kind, path)
      return Promise.resolve(`${kind}:${path}`)
    },

    getRegisteredReaders(): ReaderRegistration[] {
      return [...readers]
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

    async triggerCommand(
      commandId: string,
      note: CommandNote = { path: '', title: '', type: 'note' },
    ): Promise<void> {
      const cmd = commands.find((c) => c.id === commandId)
      if (!cmd) {
        throw new Error(`Command not registered: ${commandId}`)
      }
      await cmd.handler(note)
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
