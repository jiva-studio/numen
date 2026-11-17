import type { Randomness } from '../lib/identity'

/** The window's own randomness: no identity where the window cannot make one. */
export const browserRandomness: Randomness = {
  uuid: () =>
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : null,
}
