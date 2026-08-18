/**
 * Where a node asked for is to be drawn, declared once. Everything that
 * varies with it — the modifier that asks for it, and what a caller does about
 * it — is derived from this table.
 */

export interface ShowingDescriptor {
  /** Whether the modifier is held while asking for this one. */
  readonly modified: boolean
}

export const SHOWINGS = {
  here: { modified: false },
  beside: { modified: true },
} as const satisfies Record<string, ShowingDescriptor>

/** Where a node asked for is to be drawn: where the reader is, or beside it. */
export type PlexShowing = keyof typeof SHOWINGS

export const SHOWINGS_ALL = Object.keys(SHOWINGS) as readonly PlexShowing[]

/** One name per modifier, taken from the table. */
const BY_MODIFIER = Object.fromEntries(
  SHOWINGS_ALL.map((showing) => [SHOWINGS[showing].modified, showing]),
) as Readonly<Record<`${boolean}`, PlexShowing>>

/**
 * Which one was asked for, given the modifier held at the time. It is read
 * once, where the gesture arrives, and what travels on is the answer.
 */
export const showingOf = (modified: boolean): PlexShowing => BY_MODIFIER[`${modified}`]
