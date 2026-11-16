/** How long the window waits before it takes a stream up again, in milliseconds. */
export const AGAIN = 1000

/**
 * The message reported when a stream ended in a fault.
 * The transport fault is not actionable; what is, is that the window is behind the source and retrying.
 * One message serves every stream, so this library needs no knowledge of what is followed.
 */
export const LOST = 'lost touch with numen — the window keeps trying'
