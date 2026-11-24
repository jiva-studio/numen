/** How long the window waits before it takes a stream up again, in milliseconds. */
export const RETRY_DELAY_MS = 1000

/** Reported while a stream is down: the transport fault is not actionable, and one message serves every stream. */
export const CONNECTION_LOST_MESSAGE = 'lost touch with numen — the window keeps trying'
