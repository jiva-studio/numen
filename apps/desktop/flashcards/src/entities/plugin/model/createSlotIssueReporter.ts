import type { NoticeQueue } from '@numen/workbench'

/** Shows what is wrong with the view of a plugin in a slot as an error notice under its plugin. */
export const createSlotIssueReporter =
  (notices: Pick<NoticeQueue, 'show'>) =>
  (pluginId: string, message: string): void =>
    notices.show(pluginId, `${pluginId}: ${message}`, 'error')
