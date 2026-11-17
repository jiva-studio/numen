/** The wait between two asks while the popover is open. */
export const REFRESH_MS = 2_000

/**
 * The wait before the second ask after a change. The host answers from counts up to a second old, so
 * the ask at the change alone can return the counts from before it.
 */
export const SETTLE_MS = 2_000
