import type { NoticeHost } from '@numen/plugin-sdk'
import type { PluginContextHost, PluginHostDeps } from '../types'

/** The notices of a window that shows none: what a plugin reports is dropped. */
const DISCARDING_NOTICES: NoticeHost = { show: () => undefined, setTasks: () => undefined }

/** The services of the window as a plugin context reads them, each read at the moment a plugin asks. */
export function createServiceBridge(
  deps: PluginHostDeps,
): Pick<
  PluginContextHost,
  | 'getWorkspace'
  | 'getNotes'
  | 'getRelations'
  | 'getFiles'
  | 'getAgent'
  | 'getCommands'
  | 'getVaults'
  | 'getWindow'
  | 'getBuffers'
  | 'getNotices'
  | 'getTasks'
  | 'getBackgroundTasks'
  | 'getTexts'
> {
  return {
    getWorkspace: () => deps.getWorkspace?.(),
    getNotes: () => deps.getNotes?.(),
    getRelations: () => deps.getRelations?.(),
    getFiles: () => deps.getFiles?.(),
    getAgent: () => deps.getAgent?.(),
    getCommands: () => deps.getCommands?.(),
    getVaults: () => deps.getVaults?.(),
    getWindow: () => deps.getWindow?.(),
    getBuffers: () => deps.getBuffers?.(),
    getNotices: (pluginId) => deps.getNotices?.(pluginId) ?? DISCARDING_NOTICES,
    getTasks: (pluginId) => deps.getTasks?.(pluginId),
    getBackgroundTasks: () => deps.getBackgroundTasks?.(),
    getTexts: (pluginId) => deps.getTexts?.(pluginId),
  }
}
