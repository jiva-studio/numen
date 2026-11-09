/**
 * The plugins' tasks from the window's task list, as a plugin watches them.
 */
import { watch, type Ref } from 'vue'
import type { PluginTaskStatus, TaskHost } from '@numen/plugin-sdk'

/** One piece of work in the window's task list, as the window reports it. */
export interface WindowTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly completedCount: number
  readonly total: number
  readonly error: string
  readonly pluginId: string
  readonly pluginName: string
}

const toStatus = (task: WindowTask): PluginTaskStatus => ({
  id: task.id,
  pluginId: task.pluginId,
  pluginName: task.pluginName,
  title: task.label,
  about: task.about,
  done: task.completedCount,
  total: task.total,
  error: task.error,
})

const watchTasks = (
  tasks: Readonly<Ref<readonly WindowTask[]>>,
  isIncluded: (task: WindowTask) => boolean,
  handler: (tasks: readonly PluginTaskStatus[]) => void,
) => {
  const stop = watch(
    () => tasks.value.filter(isIncluded).map(toStatus),
    (list) => handler(list),
    { immediate: true },
  )
  return { dispose: stop }
}

/** The task host one plugin is given, over the window's one task list. */
export const createPluginTasks = (
  tasks: Readonly<Ref<readonly WindowTask[]>>,
  pluginId: string,
): TaskHost => ({
  watch: (handler) => watchTasks(tasks, (task) => task.pluginId === pluginId, handler),
  watchAll: (handler) => watchTasks(tasks, (task) => task.pluginId !== '', handler),
})
