/**
 * The notices of one plugin, in the window's one list.
 */
import type { NoticeHost } from '@numen/plugin-sdk'

/** The work a window's notice list holds for a source. */
export interface NoticeListTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly error: string
  readonly isUserStarted: boolean
  readonly done?: number
  readonly total?: number
}

/** The window's one list of notices, each held under the name of the source that gave it. */
export interface NoticeList {
  show(source: string, text: string, severity: 'info' | 'warning' | 'error'): void
  setTasks(source: string, tasks: readonly NoticeListTask[]): void
}

/** The notice host a plugin is given over the window's list. */
export const createPluginNotices = (list: NoticeList, pluginId: string): NoticeHost => ({
  show: (text, severity = 'info') => list.show(pluginId, text, severity),
  setTasks: (tasks) =>
    list.setTasks(
      pluginId,
      tasks.map((task) => ({
        id: task.id,
        label: task.label,
        about: task.about ?? '',
        error: task.error ?? '',
        isUserStarted: task.isUserStarted ?? false,
        ...(task.done === undefined ? {} : { done: task.done }),
        ...(task.total === undefined ? {} : { total: task.total }),
      })),
    ),
})
