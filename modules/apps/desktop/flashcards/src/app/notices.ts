/**
 * What the window has to say, in the corner every window says it in.
 *
 * Apart from the template because what is said, in what tone, and how long it
 * stands are rules, and a name that repeats is a notice that puts another away
 * with it.
 */
import { computed, shallowRef } from 'vue'

import { createNotice } from '@numen/ui'
import type { Notice, Task, Tone } from '@numen/ui'
import { formatErrorMessage } from '@numen/wire'

export function useNotices() {
  /** What the window is doing behind itself, which stands above what it said. */
  const tasks = shallowRef<readonly Notice[]>([])

  /** What it has told the person, newest last. */
  const told = shallowRef<readonly Notice[]>([])

  const notices = computed<readonly Notice[]>(() => [...tasks.value, ...told.value])

  /** How many have been raised, which is what names the next one. */
  let raised = 0

  const showNotice = (text: string, tone: Tone) => {
    raised += 1
    told.value = [
      ...told.value,
      {
        id: String(raised),
        says: text,
        tone,
        stay: 'kept',
        // A person pressed something and is waiting to hear. A card that waits
        // for the work to be worth drawing is a card they read ten seconds
        // late.
        isAsked: true,
      },
    ]
  }

  /**
   * Trouble, in the person's own words. A call the window itself stopped has
   * nothing to say, and nothing is raised for it.
   */
  const reportError = (why: unknown) => {
    const text = sentence(formatErrorMessage(why))
    if (text) showNotice(text, 'alarm')
  }

  /**
   * What is being done behind the window, as cards to draw. The whole list
   * arrives at once, so the whole list is what stands.
   */
  const setTasks = (work: readonly Task[]) => {
    tasks.value = work.map(createNotice)
  }

  /** One card let go of. Work put away is the corner's own to keep away. */
  const putAway = (id: string) => {
    told.value = told.value.filter((one) => one.id !== id)
  }

  return { notices, showNotice, reportError, setTasks, putAway }
}

/** One thing said, as a sentence: it opens with a capital and it ends. */
const sentence = (text: string): string => {
  const words = text.trim()
  if (!words) return ''
  const ended = /[.!?]$/.test(words) ? words : `${words}.`
  return (ended[0]?.toUpperCase() ?? '') + ended.slice(1)
}
