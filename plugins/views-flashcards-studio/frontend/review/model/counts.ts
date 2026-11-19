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
import { cards } from '../api/clients'
import { useReviewCounter } from './vault/counts'
import type { ReviewNotices } from './notices'

type VaultCountsDeps = Pick<ReviewNotices, 'reportError' | 'setReloadLost'>

async function readReloads(signal: AbortSignal, refresh: () => Promise<void>) {
  let isOpening = true
  for await (const response of cards.watchReloads({}, { signal })) {
    if (signal.aborted) return
    if (isOpening || response.reload) await refresh()
    isOpening = false
  }
}

async function followReloads(
  signal: AbortSignal,
  refresh: () => Promise<void>,
  setReloadLost: VaultCountsDeps['setReloadLost'],
) {
  const refreshCounts = async () => {
    await refresh()
    if (!signal.aborted) setReloadLost(false)
  }
  while (!signal.aborted) {
    try {
      await readReloads(signal, refreshCounts)
    } catch {
      if (!signal.aborted) setReloadLost(true)
    }
    if (signal.aborted) return
    await new Promise<void>((resolve) => setTimeout(resolve, 1000))
  }
}

export function createVaultCounts(deps: VaultCountsDeps) {
  const counter = useReviewCounter({ cards, reportError: deps.reportError })
  const abortController = new AbortController()
  let isStarted = false
  let refresh: () => Promise<void> = counter.count

  /** The counting and the following, begun once however many parts ask. */
  const start = () => {
    if (isStarted || abortController.signal.aborted) return
    isStarted = true
    counter.count().catch(deps.reportError)
    followReloads(abortController.signal, () => refresh(), deps.setReloadLost).catch(
      deps.reportError,
    )
  }

  /** What a reload does while a review is open; nothing restores counting again. */
  const setRefresh = (handler: (() => Promise<void>) | null) => {
    refresh = handler ?? counter.count
  }

  const dispose = () => {
    abortController.abort()
    deps.setReloadLost(false)
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
