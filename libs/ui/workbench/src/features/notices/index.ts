/** The notices a window is handed by the parts it hosts, and the background tasks the window lists. */
export { getTaskNotice } from './lib/notice'
export { createNoticeQueue, type NoticeQueue } from './model/queue'
export type { Notice, Stay, Task } from './lib/notice'
export type { TallyUnit, Severity } from './lib/tally'
export { createFailedTask, getFailedTasks } from './lib/backgroundTask'
export type { BackgroundTask, WorkTask } from './lib/backgroundTask'
export {
  createBackgroundTaskHost,
  type BackgroundTaskHost,
  type BackgroundTaskHostDeps,
} from './model/backgroundTasks'
