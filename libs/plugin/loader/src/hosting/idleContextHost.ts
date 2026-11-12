import type { Disposable } from '@numen/plugin-sdk'
import type {
  ContextInfoHost,
  ContextRegistrationHost,
  ContextTabHost,
  ContextVaultHost,
  ContextViewHost,
  ContextWindowHost,
  PluginContextHost,
} from './types'

const NOTHING: Disposable = { dispose: () => undefined }

const NO_TABS: ContextTabHost = {
  openTab: () => undefined,
  navigateTab: () => undefined,
  closeTab: () => undefined,
}

const NO_VAULT: ContextVaultHost = {
  getNotes: () => undefined,
  getRelations: () => undefined,
  getFiles: () => undefined,
  getBuffers: () => undefined,
  getVaults: () => undefined,
  getTexts: () => undefined,
}

const NO_WINDOW: Omit<ContextWindowHost, 'getSlots' | 'getNotices'> = {
  getWorkspace: () => undefined,
  getAgent: () => undefined,
  getCommands: () => undefined,
  isCommandListed: () => false,
  getWindow: () => undefined,
  getTasks: () => undefined,
  getBackgroundTasks: () => undefined,
  getHealth: () => undefined,
  getConnections: () => undefined,
}

/** What a window must give to host plugin contexts; everything else it leaves idle. */
export interface IdleContextHostParts {
  readonly info: ContextInfoHost
  readonly registration: Pick<ContextRegistrationHost, 'registerTabKind'>
  readonly views: Pick<ContextViewHost, 'registerView' | 'registerSlotView' | 'mountView'>
  readonly tabs?: Partial<ContextTabHost>
  readonly vault?: Partial<ContextVaultHost>
  readonly window: Pick<ContextWindowHost, 'getSlots' | 'getNotices'> & Partial<ContextWindowHost>
}

/**
 * The host of a window that has no services of the editor: nothing is read, listened to or dropped,
 * and the registrations the window has no place for are dropped. The window gives the
 * registrations it draws and the services it offers; `createPluginContext` wires the permissions.
 */
export function createIdleContextHost(parts: IdleContextHostParts): PluginContextHost {
  return {
    info: parts.info,
    registration: {
      registerReader: () => NOTHING,
      registerSettingsView: () => NOTHING,
      registerCommand: () => NOTHING,
      ...parts.registration,
    },
    views: {
      registerDropHandler: () => NOTHING,
      drops: { canDrop: () => false, drop: () => Promise.resolve([]) },
      ...parts.views,
    },
    settings: { getSetting: () => undefined, onSettingChange: () => NOTHING },
    events: {
      onVaultChange: () => NOTHING,
      onFileSelect: () => NOTHING,
      emitVaultChange: () => undefined,
      emitFileSelect: () => undefined,
    },
    tabs: { ...NO_TABS, ...parts.tabs },
    vault: { ...NO_VAULT, ...parts.vault },
    window: { ...NO_WINDOW, ...parts.window },
  }
}
