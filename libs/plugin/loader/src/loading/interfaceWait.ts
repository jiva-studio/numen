import { INTERFACE_WAIT_MS } from './limits'

/**
 * Settles when the work settles or once INTERFACE_WAIT_MS has passed, whichever is first: the moment
 * a window is drawn. A failure of the work is not this wait's to report. Once the signal aborts, the
 * wait never settles.
 */
export function waitWithinInterfaceLimit(
  work: Promise<unknown>,
  signal?: AbortSignal,
): Promise<void> {
  return new Promise<void>((resolve) => {
    if (signal?.aborted) return
    const stop = (): void => clearTimeout(timer)
    const finish = (): void => {
      stop()
      signal?.removeEventListener('abort', stop)
      if (!signal?.aborted) resolve()
    }
    const timer = setTimeout(finish, INTERFACE_WAIT_MS)
    signal?.addEventListener('abort', stop, { once: true })
    work.then(finish, finish)
  })
}
