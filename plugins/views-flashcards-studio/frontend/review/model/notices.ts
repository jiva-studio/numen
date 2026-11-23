/**
 * The notices of the window, passed to the notice host.
 *
 * Separate from the template because the message text and its severity are
 * rules: an error is shown with the failure's own message, and a call that the
 * window itself cancelled shows nothing.
 */
import { formatFailureMessage } from '../../api/failure'
import type { NoticeHost, NoticeTask, Severity } from '@numen/plugin-sdk'
export function createNotices(host: NoticeHost) {
  const showNotice = (text: string, severity: Severity) => host.show(text, severity)

  /**
   * Shows a failure message. A call that the window itself cancelled has
   * nothing to report, and no notice is raised for it.
   */
  const reportError = (error: unknown) => {
    const text = createSentence(formatFailureMessage(error))
    if (text) showNotice(text, 'error')
  }

  let work: readonly NoticeTask[] = []
  let isReloadLost = false
  const publishTasks = () => {
    const reconnect = {
      id: 'review-reload',
      label: 'Reconnecting flashcard updates',
      error: 'Lost touch with numen — the window keeps trying.',
    }
    host.setTasks(isReloadLost ? [...work, reconnect] : work)
  }
  const setTasks = (tasks: readonly NoticeTask[]) => {
    work = tasks
    publishTasks()
  }
  const setReloadLost = (lost: boolean) => {
    if (isReloadLost === lost) return
    isReloadLost = lost
    publishTasks()
  }

  return { showNotice, reportError, setTasks, setReloadLost }
}

const createSentence = (text: string): string => {
  if (!text) return ''
  const ended = /[.!?]$/.test(text) ? text : `${text}.`
  return ended.charAt(0).toUpperCase() + ended.slice(1)
}

export type ReviewNotices = ReturnType<typeof createNotices>
