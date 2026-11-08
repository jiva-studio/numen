/**
 * Background tasks with invented content for the tests and stories of the plugin.
 */
import type { BackgroundTask } from '@numen/plugin-sdk'

/** A running task with nothing counted, with the fields given replaced. */
export function createTask(fields: Partial<BackgroundTask> & { id: string }): BackgroundTask {
  return {
    label: `Index ${fields.id}`,
    about: '',
    state: 'running',
    done: 0,
    total: 0,
    counting: '',
    error: '',
    canCancel: false,
    canRetry: false,
    actionLabel: '',
    actionPluginId: '',
    ...fields,
  }
}
