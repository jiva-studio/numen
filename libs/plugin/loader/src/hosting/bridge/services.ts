import type { NoticeHost, SlotHost } from '@numen/plugin-sdk'
import type { ContextVaultHost, ContextWindowHost, PluginHostDeps } from '../types'

/** The notices of a window that shows none: what a plugin reports is dropped. */
const DISCARDING_NOTICES: NoticeHost = { show: () => undefined, setTasks: () => undefined }

/** The vault services of the window as a plugin context reads them, each read at the moment a plugin asks. */
export function createVaultServiceBridge(deps: PluginHostDeps): ContextVaultHost {
  return {
    getNotes: () => deps.getNotes?.(),
    getRelations: () => deps.getRelations?.(),
    getFiles: () => deps.getFiles?.(),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
    getBuffers: () => deps.getBuffers?.(),
    getVaults: () => deps.getVaults?.(),
  }
}

/** The services of the window itself as a plugin context reads them, each read at the moment a plugin asks. */
export function createWindowServiceBridge(
  deps: PluginHostDeps,
  slots: SlotHost,
): ContextWindowHost {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getAgent: () => deps.getAgent?.(),
    getCommands: () => deps.getCommands?.(),
    getWindow: () => deps.getWindow?.(),
    getNotices: (pluginId) => deps.getNotices?.(pluginId) ?? DISCARDING_NOTICES,
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getBackgroundTasks: () => deps.getBackgroundTasks?.(),
    getSlots: () => slots,
  }
}
