/**
 * The background tasks of the core and the plugins, as this window lists them, and the cancel of one.
 */
import { createTaskClient } from '@numen/core-client'
import { windowService } from '@/shared/api/clients'
import { WINDOW } from '@/shared/api/window'

const tasks = createTaskClient(windowService, WINDOW)

export const watchTasks = (signal: AbortSignal) => tasks.watchTasks(signal)

export const cancelTask = (id: string): Promise<void> => tasks.cancelTask(id)
