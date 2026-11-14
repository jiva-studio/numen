/** The tasks of this window as the core reports them, and the cancel of one. */
import { createTaskClient } from '@numen/core-client'
import { windowService } from '@/shared/api/clients'
import { TARGET } from '@/shared/lib/window'

const tasks = createTaskClient(windowService, TARGET)

export const watchTasks = (signal: AbortSignal) => tasks.watchTasks(signal)

export const cancelTask = (id: string): Promise<void> => tasks.cancelTask(id)
