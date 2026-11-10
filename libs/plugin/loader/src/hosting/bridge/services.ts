import type { DropHost, NoticeHost, SlotHost } from '@numen/plugin-sdk'
import type { PluginHostDeps, UiPort, VaultPort, WorkPort } from '../types'

/** The notices of a window that shows none: what a plugin reports is dropped. */
const DISCARDING_NOTICES: NoticeHost = { show: () => undefined, setTasks: () => undefined }

/** The vault services of the window as a plugin context reads them, each read at the moment a plugin asks. */
export function createVaultBridge(deps: PluginHostDeps): VaultPort {
  return {
    getNotes: () => deps.getNotes?.(),
    getRelations: () => deps.getRelations?.(),
    getFiles: () => deps.getFiles?.(),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
    getBuffers: () => deps.getBuffers?.(),
    getVaults: () => deps.getVaults?.(),
  }
}

/** The interface services of the window, with the slots and drops the host keeps itself. */
export function createUiBridge(
  deps: PluginHostDeps,
  own: { readonly slots: SlotHost; readonly drops: DropHost },
): UiPort {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getWindow: () => deps.getWindow?.(),
    getCommands: () => deps.getCommands?.(),
    getSlots: () => own.slots,
    getNotices: (pluginId) => deps.getNotices?.(pluginId) ?? DISCARDING_NOTICES,
    drops: own.drops,
  }
}

/** The agent and the background work of the window. */
export function createWorkBridge(deps: PluginHostDeps): WorkPort {
  return {
    getAgent: () => deps.getAgent?.(),
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getBackgroundTasks: () => deps.getBackgroundTasks?.(),
  }
}
