import type { Disposable } from '@numen/plugin-sdk'
import type {
  EventPort,
  PluginContextHost,
  RegistrationPort,
  SettingsPort,
  TabsPort,
  UiPort,
  VaultPort,
  WorkPort,
} from './types'

const NOTHING: Disposable = { dispose: () => undefined }

const NO_REGISTRATIONS: RegistrationPort = {
  registerReader: () => NOTHING,
  registerSettingsView: () => NOTHING,
  registerSlotView: () => NOTHING,
  registerTabKind: () => NOTHING,
  registerCommand: () => NOTHING,
  registerAgentProvider: () => NOTHING,
  registerView: () => NOTHING,
  registerDropHandler: () => NOTHING,
  mountView: () => ({ focus: () => undefined, dispose: () => undefined }),
}

const NO_SETTINGS: SettingsPort = { getSetting: () => undefined, onSettingChange: () => NOTHING }

const NO_EVENTS: EventPort = {
  onVaultChange: () => NOTHING,
  onFileSelect: () => NOTHING,
  emitVaultChange: () => undefined,
  emitFileSelect: () => undefined,
}

const NO_TABS: TabsPort = {
  openTab: () => undefined,
  navigateTab: () => undefined,
  closeTab: () => undefined,
}

const NO_VAULT: VaultPort = {
  getNotes: () => undefined,
  getRelations: () => undefined,
  getFiles: () => undefined,
  getTexts: () => undefined,
  getBuffers: () => undefined,
  getVaults: () => undefined,
}

const NO_WORK: WorkPort = {
  getAgent: () => undefined,
  getTasks: () => undefined,
  getBackgroundTasks: () => undefined,
}

const NO_DROPS: UiPort['drops'] = { canDrop: () => false, drop: () => Promise.resolve([]) }

/**
 * What a window with no services of the editor gives: the ports it draws and the services it has.
 * Every service not given is a port that does nothing.
 */
export interface IdleContextHostParts {
  readonly app: PluginContextHost['app']
  readonly target: PluginContextHost['target']
  readonly openExternal: PluginContextHost['openExternal']
  readonly registrations: Pick<
    RegistrationPort,
    'registerTabKind' | 'registerView' | 'registerSlotView' | 'mountView'
  >
  readonly ui: Pick<UiPort, 'getSlots' | 'getNotices'> & Partial<UiPort>
  readonly tabs?: Partial<TabsPort>
  readonly vault?: Partial<VaultPort>
  readonly work?: Partial<WorkPort>
}

/**
 * The host of a window that has no services of the editor: nothing is read, listened to or dropped,
 * and the registrations the window has no place for are dropped. The window gives the
 * registrations it draws and the services it offers; `createPluginContext` wires the permissions.
 */
export function createIdleContextHost(parts: IdleContextHostParts): PluginContextHost {
  return {
    app: parts.app,
    target: parts.target,
    openExternal: parts.openExternal,
    registrations: { ...NO_REGISTRATIONS, ...parts.registrations },
    settings: NO_SETTINGS,
    events: NO_EVENTS,
    tabs: { ...NO_TABS, ...parts.tabs },
    vault: { ...NO_VAULT, ...parts.vault },
    ui: {
      getWorkspace: () => undefined,
      getWindow: () => undefined,
      getCommands: () => undefined,
      drops: NO_DROPS,
      ...parts.ui,
    },
    work: { ...NO_WORK, ...parts.work },
  }
}
