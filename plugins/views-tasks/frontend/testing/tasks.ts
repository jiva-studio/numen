/**
 * Background tasks with invented content for the tests and stories of the plugin.
 */
import type { BackgroundTask } from '@numen/plugin-sdk'

/** What a test sets of a task: the fields given replace the defaults, and each group replaces them field by field. */
export interface TaskFields extends Partial<
  Omit<BackgroundTask, 'progress' | 'allowed' | 'action'>
> {
  readonly id: string
  readonly progress?: Partial<BackgroundTask['progress']>
  readonly allowed?: Partial<BackgroundTask['allowed']>
  readonly action?: Partial<BackgroundTask['action']>
}

/** A running task with nothing counted, with the fields given replaced. */
export function createTask({ progress, allowed, action, ...fields }: TaskFields): BackgroundTask {
  return {
    label: `Index ${fields.id}`,
    about: '',
    state: 'running',
    error: '',
    ...fields,
    progress: { completedCount: 0, total: 0, counting: '', ...progress },
    allowed: { cancel: false, retry: false, ...allowed },
    action: { label: '', ...action },
  }
}
