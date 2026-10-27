/** What a window has to say while it works: notices, and the counts in them. */
export { default as Notices } from './ui/Notices.vue'
export { createNotice } from './lib/notice'
export { createNoticeQueue, type NoticeQueue } from './model/queue'
export type { Notice, Stay, Task } from './lib/notice'
export type { TallyUnit, Severity } from './lib/tally'
