/**
 * What the window has to say, in the corner every window says it in.
 *
 * Apart from the template because what is said, in what tone, and how long it
 * stands are rules, and a name that repeats is a notice that puts another away
 * with it.
 */
import { computed, ref } from 'vue'
import { ConnectError } from '@connectrpc/connect'

import type { Notice, Tone } from '@numen/ui'

/** One piece of work the window is doing behind itself, as the answer holds it. */
export interface Task {
  readonly id: string
  readonly doing: string
  readonly about: string
  readonly failed: string
  readonly asked: boolean
}

export function raising() {
  /** What the window is doing behind itself, which stands above what it said. */
  const working = ref<readonly Notice[]>([])

  /** What it has told the person, newest last. */
  const told = ref<readonly Notice[]>([])

  const notices = computed<readonly Notice[]>(() => [...working.value, ...told.value])

  /** How many have been raised, which is what names the next one. */
  let raised = 0

  const says = (said: string, tone: Tone) => {
    raised += 1
    told.value = [
      ...told.value,
      {
        id: String(raised),
        says: said,
        tone,
        stay: 'kept',
        // A person pressed something and is waiting to hear. A card that waits
        // for the work to be worth drawing is a card they read ten seconds
        // late.
        asked: true,
      },
    ]
  }

  /** Trouble, in the person's own words: what the application said, as a sentence. */
  const failed = (why: unknown) => says(sentence(ConnectError.from(why).rawMessage), 'alarm')

  /**
   * What is being done behind the window, as cards to draw. The whole list
   * arrives at once, so the whole list is what stands.
   */
  const doing = (tasks: readonly Task[]) => {
    working.value = tasks.map((at) => ({
      id: at.id,
      says: at.failed || at.doing,
      about: at.about,
      working: at.failed === '',
      asked: at.asked || at.failed !== '',
      ...(at.failed ? { tone: 'alarm' as const, stay: 'kept' as const } : {}),
    }))
  }

  /** One card let go of. Work put away is the corner's own to keep away. */
  const putAway = (id: string) => {
    told.value = told.value.filter((one) => one.id !== id)
  }

  return { notices, says, failed, doing, putAway }
}

/** One thing said, as a sentence: it opens with a capital and it ends. */
const sentence = (said: string): string => {
  const words = said.trim()
  if (!words) return ''
  const ended = /[.!?]$/.test(words) ? words : `${words}.`
  return (ended[0]?.toUpperCase() ?? '') + ended.slice(1)
}
