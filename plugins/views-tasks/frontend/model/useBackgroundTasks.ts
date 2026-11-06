/**
 * The background tasks of the window as a reactive list, for as long as the component that reads
 * them is mounted.
 */
import { onScopeDispose, shallowRef, type ShallowRef } from 'vue'
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-sdk'

export function useBackgroundTasks(
  host: BackgroundTaskHost,
): Readonly<ShallowRef<readonly BackgroundTask[]>> {
  const tasks = shallowRef<readonly BackgroundTask[]>([])
  const watching = host.watch((list) => {
    tasks.value = list
  })
  onScopeDispose(() => watching.dispose())
  return tasks
}
