/**
 * What the window has to say, in the corner every window says it in.
 *
 * Apart from the template because what is said, in what tone, and how long it
 * stands are rules, and a name that repeats is a notice that puts another away
 * with it.
 */
import { ref } from 'vue'

import type { Notice, Tone } from '@numen/ui'

export function saying() {
  const notices = ref<readonly Notice[]>([])

  /** How many have been raised, which is what names the next one. */
  let raised = 0

  const says = (said: string, tone: Tone) => {
    raised += 1
    notices.value = [
      ...notices.value,
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

  /** Trouble, in the person's own words. */
  const failed = (why: unknown) => says(String(why), 'alarm')

  const putAway = (id: string) => {
    notices.value = notices.value.filter((one) => one.id !== id)
  }

  return { notices, says, failed, putAway }
}
