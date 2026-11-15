/** The notices a window is handed by the parts it hosts, and the background tasks the window lists. */
export { createNoticeQueue, type NoticeQueue } from './model/queue'
export { createFailedTask, getFailedTasks } from './lib/backgroundTask'
export type { WorkTask } from './lib/backgroundTask'
export { createBackgroundTaskHost } from './model/backgroundTasks'
