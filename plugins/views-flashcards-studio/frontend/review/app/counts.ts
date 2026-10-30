/**
 * The cards due in every vault today, counted once for the window and read by
 * each part of the review that shows them: the summary beside each vault on the
 * window's list, and the decks of the open vault.
 *
 * Counting starts with the first part that requests it, and runs until the
 * plugin is disposed. A deck written or a card changed outside the window is
 * tracked for as long, and handled by the refresh that the open review sets, or
 * by counting again where no review is open.
 */
import { createFollower } from '@numen/ui'

import { cards } from '../shared/clients'
import { useReviewCounter } from '../entities/vault'

export interface VaultCountsDeps {
  /** Reports a failure to the user. */
  readonly reportError: (why: unknown) => void
}

export function createVaultCounts(deps: VaultCountsDeps) {
  const counter = useReviewCounter({ cards, reportError: deps.reportError })
  const abortController = new AbortController()
  let isOpen = true
  let isStarted = false
  let refresh: () => Promise<void> = counter.count

  const follow = createFollower({
    isOpen: () => isOpen,
    setLost: deps.reportError,
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  /** The counting and the following, begun once however many parts ask. */
  const start = () => {
    if (isStarted || !isOpen) return
    isStarted = true
    counter.count().catch(deps.reportError)
    follow(
      () => cards.watchReloads({}, { signal: abortController.signal }),
      async (response) => {
        // A message carrying no reload keeps the stream open and moves nothing.
        if (!response.reload) return
        await refresh()
      },
    ).catch(deps.reportError)
  }

  /** What a reload does while a review is open; nothing restores counting again. */
  const setRefresh = (handler: (() => Promise<void>) | null) => {
    refresh = handler ?? counter.count
  }

  const dispose = () => {
    isOpen = false
    abortController.abort()
    counter.stop()
  }

  return {
    vaults: counter.vaults,
    isCounting: counter.isCounting,
    day: counter.day,
    count: counter.count,
    stop: counter.stop,
    start,
    setRefresh,
    dispose,
  }
}

export type VaultCounts = ReturnType<typeof createVaultCounts>
