/** The tasks of this window as the core reports them, and the cancel of one. */
import { createTaskClient } from '@numen/core-client'
import type { WindowClients } from '@/shared/api/clients'
import { TARGET } from '@/shared/lib/window'

/** The tasks the core runs for this window, followed as a stream, and the cancel of one. */
export const createTaskApi = (windowService: WindowClients['windowService']) => {
  const tasks = createTaskClient(windowService, TARGET)
  return {
    watchTasks: (signal: AbortSignal) => tasks.watchTasks(signal),
    cancelTask: (id: string): Promise<void> => tasks.cancelTask(id),
  }
}

export type TaskApi = ReturnType<typeof createTaskApi>

export { createStreamReader } from '@numen/core-client'
