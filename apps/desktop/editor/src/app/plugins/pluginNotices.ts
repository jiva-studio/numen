/**
 * The notices of one plugin, in the window's one list.
 */
import type { NoticeHost } from '@numen/plugin-sdk'
import type { NoticeQueue } from '@numen/shell'

export const createPluginNotices = (queue: NoticeQueue, pluginId: string): NoticeHost => ({
  show: (text, tone = 'plain') => queue.show(pluginId, text, tone),
  setTasks: (tasks) =>
    queue.setTasks(
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
