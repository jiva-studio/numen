/**
 * The draft of the connection the link inspector edits, and the rows it shows for it. Each
 * function returns a new draft; which roles the two records carry is the core's to decide.
 */
import type { Connection, ConnectionRole, ConnectionSide } from '@numen/plugin-sdk'

/** Which of the two records of a connection: the one the first note writes, or the second. */
export type SideKey = 'first' | 'second'

/**
 * Which way a row of the inspector points: `none` draws no arrow, `forward` points from the first
 * note to the second, and `backward` from the second note to the first.
 */
export type LinkDirection = 'none' | 'forward' | 'backward'

/** One row of the inspector: one record of the connection and the way it points. */
export interface ConnectionRow {
  readonly key: SideKey
  readonly side: ConnectionSide
  readonly direction: LinkDirection
}

/** The role a connection takes when a record is added to one that has none. */
export const NEW_LINK_ROLE: ConnectionRole = 'jump'

const BLANK_SIDE: ConnectionSide = { label: '', marker: null, reason: '' }

const getSide = (draft: Connection, key: SideKey): ConnectionSide | null =>
  key === 'first' ? draft.firstSide : draft.secondSide

const withSide = (draft: Connection, key: SideKey, side: ConnectionSide | null): Connection =>
  key === 'first' ? { ...draft, firstSide: side } : { ...draft, secondSide: side }

const getDirection = (key: SideKey, side: ConnectionSide): LinkDirection => {
  if (side.marker !== 'arrow') return 'none'
  return key === 'first' ? 'forward' : 'backward'
}

/** The rows the inspector shows: the record of the first note, then that of the second. */
export function getRows(draft: Connection): ConnectionRow[] {
  const rows: ConnectionRow[] = []
  if (draft.firstSide) {
    rows.push({
      key: 'first',
      side: draft.firstSide,
      direction: getDirection('first', draft.firstSide),
    })
  }
  if (draft.secondSide) {
    rows.push({
      key: 'second',
      side: draft.secondSide,
      direction: getDirection('second', draft.secondSide),
    })
  }
  return rows
}

/** The direction a row turns to when its button is pressed: the three in turn for a lone record, arrow or none for one of two. */
export function getNextDirection(
  direction: LinkDirection,
  key: SideKey,
  rowCount: number,
): LinkDirection {
  if (rowCount > 1) {
    if (direction !== 'none') return 'none'
    return key === 'first' ? 'forward' : 'backward'
  }
  switch (direction) {
    case 'none':
      return 'forward'
    case 'forward':
      return 'backward'
    case 'backward':
      return 'none'
  }
}

/**
 * The draft with one record turned to its next direction. A lone record goes none, forward,
 * backward and none again: forward and none are written by the first note and backward by the
 * second, so the record moves between the notes with its label. Of two records each one only
 * gains or loses its own arrow, so none of them moves. A draft without that record is returned
 * as it is.
 */
export function cycleDirection(draft: Connection, key: SideKey): Connection {
  const rows = getRows(draft)
  const row = rows.length > 1 ? rows.find((candidate) => candidate.key === key) : rows[0]
  if (!row) return draft
  const { side } = row
  if (rows.length > 1) {
    return withSide(draft, key, { ...side, marker: row.direction === 'none' ? 'arrow' : null })
  }
  switch (getNextDirection(row.direction, row.key, 1)) {
    case 'forward':
      return { ...draft, firstSide: { ...side, marker: 'arrow' }, secondSide: null }
    case 'backward':
      return { ...draft, firstSide: null, secondSide: { ...side, marker: 'arrow' } }
    default:
      return { ...draft, firstSide: { ...side, marker: null }, secondSide: null }
  }
}

/**
 * The draft with a record added. A draft with no record gets an undirected one in the first note;
 * one with a single record gets the record of the other note with an empty label, and both point
 * at the note they are about.
 */
export function addReverseSide(draft: Connection): Connection {
  const role = draft.role ?? NEW_LINK_ROLE
  const { firstSide, secondSide } = draft
  if (firstSide && secondSide) return draft
  if (!firstSide && !secondSide) return { ...draft, role, firstSide: BLANK_SIDE }
  return {
    ...draft,
    role,
    firstSide: { ...(firstSide ?? BLANK_SIDE), marker: 'arrow' },
    secondSide: { ...(secondSide ?? BLANK_SIDE), marker: 'arrow' },
  }
}

/** The draft without the record of one note. */
export const removeSide = (draft: Connection, key: SideKey): Connection =>
  withSide(draft, key, null)

/** The draft with the label of one record replaced. A note that writes no record has no label. */
export function setSideLabel(draft: Connection, key: SideKey, label: string): Connection {
  const side = getSide(draft, key)
  return side === null ? draft : withSide(draft, key, { ...side, label })
}

/** Whether neither note writes a record. */
export const isConnectionEmpty = (draft: Connection): boolean =>
  draft.firstSide === null && draft.secondSide === null

const isSideEqual = (one: ConnectionSide | null, other: ConnectionSide | null): boolean =>
  one === null || other === null
    ? one === other
    : one.label === other.label && one.marker === other.marker && one.reason === other.reason

/** Whether two drafts say the same thing. */
export const areConnectionsEqual = (one: Connection, other: Connection): boolean =>
  one.role === other.role &&
  one.type === other.type &&
  isSideEqual(one.firstSide, other.firstSide) &&
  isSideEqual(one.secondSide, other.secondSide)
