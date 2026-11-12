import type { NoticeQueue } from '@numen/workbench'

/** Shows an issue of a view of a plugin as an error of that plugin, led by its id. */
export const createViewIssueReporter =
  (notices: Pick<NoticeQueue, 'show'>) =>
  (pluginId: string, message: string): void =>
    notices.show(pluginId, `${pluginId}: ${message}`, 'error')
