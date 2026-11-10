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

/** What a window must give to host plugin contexts; everything else it leaves idle. */
export interface IdleContextHostParts {
  readonly info: ContextInfoHost
  readonly registration: Pick<ContextRegistrationHost, 'registerTabKind'>
  readonly views: Pick<ContextViewHost, 'registerView' | 'registerSlotView' | 'mountView'>
  readonly tabs?: ContextTabHost
  readonly vault?: ContextVaultHost
  readonly window?: ContextWindowHost
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
      registerAgentProvider: () => NOTHING,
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
    tabs: parts.tabs ?? {},
    vault: parts.vault ?? {},
    window: parts.window ?? {},
  }
}
