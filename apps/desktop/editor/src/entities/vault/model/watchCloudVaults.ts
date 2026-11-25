/**
 * Keeps the cloud vaults fresh while the window is shown: a refresh when the window gets focus or
 * becomes visible, and one every interval while it stays visible.
 */
import { win } from '@numen/ui'

export const CLOUD_REFRESH_INTERVAL_MS = 5 * 60 * 1000

/** What watching the window uses of the browser. */
export interface CloudWatchPorts {
  readonly isVisible: () => boolean
  /** Calls `listener` at every focus of the window and every change of its visibility; returns the stop. */
  readonly onWake: (listener: () => void) => () => void
  /** Calls `task` every `ms` milliseconds; returns the stop. */
  readonly every: (task: () => void, ms: number) => () => void
}

export const browserCloudWatchPorts: CloudWatchPorts = {
  isVisible: () => win.document.visibilityState === 'visible',
  onWake: (listener) => {
    win.addEventListener('focus', listener)
    win.document.addEventListener('visibilitychange', listener)
    return () => {
      win.removeEventListener('focus', listener)
      win.document.removeEventListener('visibilitychange', listener)
    }
  },
  every: (task, ms) => {
    const timer = setInterval(task, ms)
    return () => clearInterval(timer)
  },
}

/** Refreshes through `refresh`, which does not overlap itself; returns the stop of the watching. */
export function watchCloudVaults(
  refresh: () => void,
  ports: CloudWatchPorts = browserCloudWatchPorts,
): () => void {
  let stopTimer: (() => void) | undefined

  const startTimer = (): void => {
    stopTimer ??= ports.every(refresh, CLOUD_REFRESH_INTERVAL_MS)
  }
  const endTimer = (): void => {
    stopTimer?.()
    stopTimer = undefined
  }
  const wake = (): void => {
    if (!ports.isVisible()) return endTimer()
    startTimer()
    refresh()
  }

  if (ports.isVisible()) startTimer()
  const stopWake = ports.onWake(wake)
  return () => {
    stopWake()
    endTimer()
  }
}
