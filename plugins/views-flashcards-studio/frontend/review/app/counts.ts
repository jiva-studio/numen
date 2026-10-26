/**
 * What every vault owes today, counted once for the window and read by each part of the review that
 * shows it: the summary beside each vault on the window's list, and the decks of the vault open.
 *
 * Counting starts with the first part that asks for it, and runs until the plugin is disposed. A
 * deck written or a card changed underneath the window is followed for as long, and answered by
 * whatever refresh the open review sets, or by counting again where none is open.
 */
import { createFollower } from '@numen/ui'

import { cards } from '../shared/clients'
import { useReviewCounter } from '../entities/vault'

export interface VaultCountsDeps {
  /** What went wrong, in the person's own words. */
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
    void counter.count()
    void follow(
      () => cards.watchReloads({}, { signal: abortController.signal }),
      async (response) => {
        // A message carrying no reload keeps the stream open and moves nothing.
        if (!response.reload) return
        await refresh()
      },
    )
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
