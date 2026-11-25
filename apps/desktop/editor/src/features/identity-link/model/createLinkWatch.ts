/**
 * The link the core holds, watched: the reads of it, and what ends it. A link that stalls is cancelled in the core
 * and reported through `isStalled`.
 */
import type { Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { LinkInfo } from '@/entities/identity'
import { isEnded } from '../lib/phases'
import type { LinkCalls } from '../lib/types'
import { createLinkPoll } from './createLinkPoll'

const POLL_MS = 500

export function createLinkWatch(
  getCalls: () => LinkCalls,
  link: Ref<LinkInfo | undefined>,
  isStalled: Ref<boolean>,
  stallMs: number,
) {
  const cancel = async (): Promise<void> => {
    poll.stop()
    if (link.value !== undefined && !isEnded(link.value)) await getCalls().cancel()
  }

  const poll = createLinkPoll({
    read: () => getCalls().get(),
    isEnded,
    intervalMs: POLL_MS,
    run: (task) => runInBackground(task, () => undefined),
    isWaiting: (info) => info.phase === 'waitingForChannel',
    stallAfter: Math.ceil(stallMs / POLL_MS),
    onStalled: () => {
      isStalled.value = true
      runInBackground(cancel(), () => undefined)
    },
    onInfo: (info) => {
      link.value = info
    },
  })

  return { poll, cancel }
}
