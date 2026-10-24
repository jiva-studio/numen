/**
 * The latest request wins.
 *
 * Several responses can be in flight at once. A caller draws the response to the
 * request last initiated, or processes responses in the order they land and never
 * goes back.
 */

/** One request in flight, measured against everything requested after it. */
export interface InFlightRequest {
  /** Whether the response to this one is still the response to process. */
  readonly isCurrent: boolean
  /**
   * Whether this response may be applied over what is already applied. A response
   * older than one that has completed is discarded.
   */
  claim(): boolean
}

export type RequestGuard = ReturnType<typeof createRequestGuard>

export function createRequestGuard() {
  let requestSeq = 0
  let processedSeq = 0
  let isListening = true

  /** Acquire in-flight token for a request. */
  const acquireRequest = (): InFlightRequest => {
    const mine = ++requestSeq
    return {
      get isCurrent() {
        return isListening && mine === requestSeq
      },
      claim() {
        if (!isListening || mine < processedSeq) return false
        processedSeq = mine
        return true
      },
    }
  }

  /** Cancel all existing requests so that nothing in flight will be processed. */
  const cancel = () => {
    requestSeq += 1
    processedSeq = requestSeq
  }

  /** Nothing will be processed from here on. */
  const close = () => {
    isListening = false
  }

  /** Whether requests are still being processed. */
  const open = () => isListening

  return { acquireRequest, cancel, close, open }
}
