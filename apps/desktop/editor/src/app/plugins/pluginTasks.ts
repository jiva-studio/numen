/**
 * The plugins' tasks from the window's task list, as a plugin watches them.
 */
import { watch, type Ref } from 'vue'
import type { PluginTaskStatus, TaskHost } from '@numen/plugin-sdk'
import type { Task } from '@/shared/notices/task'

const toStatus = (task: Task): PluginTaskStatus => ({
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
  tasks: Readonly<Ref<readonly Task[]>>,
  isIncluded: (task: Task) => boolean,
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
  tasks: Readonly<Ref<readonly Task[]>>,
  pluginId: string,
): TaskHost => ({
  watch: (handler) => watchTasks(tasks, (task) => task.pluginId === pluginId, handler),
  watchAll: (handler) => watchTasks(tasks, (task) => task.pluginId !== '', handler),
})
