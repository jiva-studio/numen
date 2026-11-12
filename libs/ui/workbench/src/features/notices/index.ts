/** The notices a window is handed by the parts it hosts, before the window turns them into tasks. */
export { getTaskNotice } from './lib/notice'
export { createNoticeQueue, type NoticeQueue } from './model/queue'
export type { Notice, Stay, Task } from './lib/notice'
export type { TallyUnit, Severity } from './lib/tally'
