/**
 * Frontend entrypoint for the Tasks plugin: the summary of the background tasks of the window and
 * their list, drawn in the slot `titleBar.status`.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type BackgroundTaskHost, type PluginContext } from '@numen/plugin-sdk'
import { useBackgroundTasks } from './model/useBackgroundTasks'
import TasksStatus from './ui/TasksStatus.vue'

/** The id the manifest places the view of the tasks under. */
export const TASKS_VIEW = 'status'

/** The view of the slot: the summary and the list over the background tasks of the host. */
export function createTasksView(host: BackgroundTaskHost | undefined) {
  return defineComponent({
    name: 'TasksView',
    setup: () => {
      if (!host) return () => null
      const tasks = useBackgroundTasks(host)
      return () =>
        h(TasksStatus, {
          tasks: tasks.value,
          // The host lists a task it did not cancel as running again and reports why.
          onCancel: (id: string) => host.cancel(id).catch(() => undefined),
          onRetry: (id: string) => host.retry(id),
          onDismiss: (id: string) => host.dismiss(id),
          onOpenSettings: (pluginId: string) => host.openSettings(pluginId),
        })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.register.slotView({
    id: TASKS_VIEW,
    component: createTasksView(context.work.backgroundTasks),
  })
}

export default definePlugin({
  activate,
})
