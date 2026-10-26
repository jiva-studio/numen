/**
 * What the window has to say, handed to the corner the host draws.
 *
 * Apart from the template because what is said and in what tone are rules: an
 * error is said in the person's own words, and a call the window itself
 * stopped says nothing.
 */
import type { NoticeHost, NoticeTask, NoticeTone } from '@numen/plugin-sdk'
import { formatErrorMessage } from '@numen/wire'

export function createNotices(host: NoticeHost) {
  const showNotice = (text: string, tone: NoticeTone) => host.show(text, tone)

  /**
   * Trouble, in the person's own words. A call the window itself stopped has
   * nothing to say, and nothing is raised for it.
   */
  const reportError = (why: unknown) => {
    const text = sentence(formatErrorMessage(why))
    if (text) showNotice(text, 'alarm')
  }

  /** What is being done behind the window. The whole list arrives at once. */
  const setTasks = (work: readonly NoticeTask[]) => host.setTasks(work)

  return { showNotice, reportError, setTasks }
}

/** One thing said, as a sentence: it opens with a capital and it ends. */
const sentence = (text: string): string => {
  if (!text) return ''
  const ended = /[.!?]$/.test(text) ? text : `${text}.`
  return ended.charAt(0).toUpperCase() + ended.slice(1)
}
