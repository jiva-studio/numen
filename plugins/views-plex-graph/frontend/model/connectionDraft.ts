/**
 * The draft of the connection the link inspector edits. Each function returns the
 * draft with one field changed; which roles the two records carry is the core's to decide.
 */
import type { Connection, ConnectionRole, ConnectionSide } from '@numen/plugin-sdk'

/** Which of the two records of a connection: the one the first note writes, or the second. */
export type SideKey = 'first' | 'second'

/** The role a connection takes when a record is added to one that has none. */
export const NEW_LINK_ROLE: ConnectionRole = 'jump'

/** A record that is present and says nothing: no label and no marker. */
const BLANK_SIDE: ConnectionSide = { label: '', marker: null, reason: '' }

/** The record one note writes in a draft, and null where it writes none. */
export const getSide = (draft: Connection, key: SideKey): ConnectionSide | null =>
  key === 'first' ? draft.firstSide : draft.secondSide

const withSide = (draft: Connection, key: SideKey, side: ConnectionSide | null): Connection =>
  key === 'first' ? { ...draft, firstSide: side } : { ...draft, secondSide: side }

/** The draft with the record of one note added or taken away. */
export function setSidePresent(draft: Connection, key: SideKey, isPresent: boolean): Connection {
  if (!isPresent) return withSide(draft, key, null)
  if (getSide(draft, key) !== null) return draft
  return { ...withSide(draft, key, BLANK_SIDE), role: draft.role ?? NEW_LINK_ROLE }
}

/** The draft with the label of one record replaced. A note that writes no record has no label. */
export function setSideLabel(draft: Connection, key: SideKey, label: string): Connection {
  const side = getSide(draft, key)
  return side === null ? draft : withSide(draft, key, { ...side, label })
}

/** The draft with the arrow of one record switched on or off. The label stays as it is. */
export function toggleSideArrow(draft: Connection, key: SideKey): Connection {
  const side = getSide(draft, key)
  if (side === null) return draft
  return withSide(draft, key, { ...side, marker: side.marker === 'arrow' ? null : 'arrow' })
}

/** The draft with the kind of link replaced. */
export const setRole = (draft: Connection, role: ConnectionRole): Connection => ({
  ...draft,
  role,
})

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
