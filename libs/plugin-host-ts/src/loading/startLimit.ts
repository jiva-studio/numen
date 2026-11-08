/**
 * The limit of one plugin start. It runs from the moment the start begins, the read of the module
 * of the plugin included, and passes after PLUGIN_START_LIMIT_MS.
 */
import { PLUGIN_START_LIMIT_MS } from './limits'

export interface StartLimit {
  /** Settles with 'expired' once the limit has passed. */
  readonly expired: Promise<'expired'>
  readonly isPassed: () => boolean
  /** Stops the timer of a limit the start no longer needs. */
  readonly stop: () => void
}

export function createStartLimit(): StartLimit {
  let isPassed = false
  let timer: ReturnType<typeof setTimeout> | undefined
  const expired = new Promise<'expired'>((resolve) => {
    timer = setTimeout(() => {
      isPassed = true
      resolve('expired')
    }, PLUGIN_START_LIMIT_MS)
  })
  return { expired, isPassed: () => isPassed, stop: () => clearTimeout(timer) }
}
