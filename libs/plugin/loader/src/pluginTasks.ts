/**
 * The plugins' tasks from the window's task list, as a plugin watches them.
 */
import { watch, type Ref } from 'vue'
import type { CoreTask } from '@numen/core-client'
import type { PluginTaskStatus, TaskHost } from '@numen/plugin-sdk'

/** One piece of work in the window's task list, as the window reports it. */
type WindowTask = Pick<CoreTask, 'id' | 'label' | 'about' | 'error'> & {
  readonly progress: Pick<CoreTask['progress'], 'completedCount' | 'total'>
  readonly source: Pick<CoreTask['source'], 'pluginId' | 'pluginName'>
}

const toStatus = (task: WindowTask): PluginTaskStatus => ({
  id: task.id,
  pluginId: task.source.pluginId,
  pluginName: task.source.pluginName,
  title: task.label,
  about: task.about,
  done: task.progress.completedCount,
  total: task.progress.total,
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
  watch: (handler) => watchTasks(tasks, (task) => task.source.pluginId === pluginId, handler),
  watchAll: (handler) => watchTasks(tasks, (task) => task.source.pluginId !== '', handler),
})
