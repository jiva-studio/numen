import type { Disposable } from '@numen/plugin-sdk'
import type { PluginContextHost } from './types'

const NOTHING: Disposable = { dispose: () => undefined }

/** What a window must give to host plugin contexts; everything else it leaves idle. */
export type IdleContextHostParts = Pick<
  PluginContextHost,
  'app' | 'openExternal' | 'registerTabKind' | 'registerView' | 'registerSlotView' | 'mountView'
> &
  Partial<PluginContextHost>

/**
 * The host of a window that has no services of the editor: nothing is read, listened to or dropped,
 * and the registrations the window has no place for are kept by nobody. The window gives the
 * registrations it draws and the services it offers; `createPluginContext` wires the permissions.
 */
export function createIdleContextHost(parts: IdleContextHostParts): PluginContextHost {
  return {
    registerReader: () => NOTHING,
    registerSettingsView: () => NOTHING,
    registerCommand: () => NOTHING,
    registerAgentProvider: () => NOTHING,
    registerDropHandler: () => NOTHING,
    drops: { canDrop: () => false, drop: () => Promise.resolve([]) },
    getSetting: () => undefined,
    onSettingChange: () => NOTHING,
    onVaultChange: () => NOTHING,
    onFileSelect: () => NOTHING,
    emitVaultChange: () => undefined,
    emitFileSelect: () => undefined,
    ...parts,
  }
}
