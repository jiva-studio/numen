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
  const reportError = (why: unknown) => {
    const text = sentence(formatFailureMessage(why))
    if (text) showNotice(text, 'error')
  }

  /** What is being done behind the window. The whole list arrives at once. */
  const setTasks = (work: readonly NoticeTask[]) => host.setTasks(work)

  return { showNotice, reportError, setTasks }
}

/** One message as a sentence: it starts with a capital letter and ends with punctuation. */
const sentence = (text: string): string => {
  if (!text) return ''
  const ended = /[.!?]$/.test(text) ? text : `${text}.`
  return ended.charAt(0).toUpperCase() + ended.slice(1)
}
