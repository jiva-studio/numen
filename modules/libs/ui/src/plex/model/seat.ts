/**
 * Every seat a node can take, declared once. Everything that varies with a
 * seat — directions, wording, colour — is derived from this table.
 */

export interface SeatDescriptor {
  readonly grows: 'up' | 'down' | 'left' | 'right'
  readonly plural: readonly [one: string, many: string]
}

export const SEATS = {
  parent: { grows: 'up', plural: ['parent', 'parents'] },
  child: { grows: 'down', plural: ['child', 'children'] },
  jump: { grows: 'left', plural: ['jump', 'jumps'] },
  sibling: { grows: 'right', plural: ['sibling', 'siblings'] },
} as const satisfies Record<string, SeatDescriptor>

/** Every seat except the one that may appear only once. */
export type PlexRelatedSeat = keyof typeof SEATS

/** Where a node sits relative to the one in focus. */
export type PlexSeat = 'focus' | PlexRelatedSeat

export const RELATED_SEATS = Object.keys(SEATS) as readonly PlexRelatedSeat[]

/** One seat, by name: `parent`, `jump`. */
export const seatWord = (seat: PlexRelatedSeat): string => SEATS[seat].plural[0]

/** `3 children`, `1 jump`. */
export const countOf = (seat: PlexRelatedSeat, count: number): string =>
  `${count} ${SEATS[seat].plural[count === 1 ? 0 : 1]}`
