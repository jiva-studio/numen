import { windowService } from '@/shared/api/clients'
import { TARGET } from '@/shared/lib/window'
import type { WindowTask } from '../lib/task'

/** The tasks of this window as the core reports them, a whole list at each change, until the signal aborts. */
export async function* watchTasks(signal: AbortSignal): AsyncGenerator<readonly WindowTask[]> {
  for await (const response of windowService.watchTasks({ window: TARGET }, { signal })) {
    yield response.tasks.map((task) => ({
      id: task.id,
      label: task.title,
      about: task.about,
      error: task.error,
      isUserStarted: task.isUserStarted,
    }))
  }
}
