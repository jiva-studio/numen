/**
 * Relationship conversions between schema representations and the SDK models.
 */
import {
  NOTE_MARKER,
  NOTE_ROLE,
  findErrorCode,
  getNames,
  type Backlink,
  type Connection,
  type ConnectionRole,
  type ConnectionSide,
  type LinkMarker,
  type LinkRole,
  type NoteMarkerCode,
  type NoteRoleCode,
  type RelationError,
} from '@numen/plugin-sdk'

/** The role of a link, as named in the window. */
const calledRole: Record<NoteRoleCode, LinkRole | null> = {
  [NOTE_ROLE.UNSPECIFIED]: null,
  [NOTE_ROLE.PARENT]: 'parent',
  [NOTE_ROLE.CHILD]: 'child',
  [NOTE_ROLE.JUMP]: 'jump',
  [NOTE_ROLE.REF]: 'ref',
  [NOTE_ROLE.ATTACHMENT]: 'attachment',
}

/** The role of a link, as named in the schema. */
export const roleCodes = getNames<LinkRole, NoteRoleCode>(calledRole)

/** What a link draws at its target, as named in the window. */
const calledMarker: Record<NoteMarkerCode, LinkMarker | null> = {
  [NOTE_MARKER.UNSPECIFIED]: null,
  [NOTE_MARKER.ARROW]: 'arrow',
}

/** What a link draws at its target, as named in the schema. */
const markerCodes = getNames<LinkMarker, NoteMarkerCode>(calledMarker)

const connectionRoles: Record<ConnectionRole, true> = { parent: true, child: true, jump: true }

/** The roles a note can be created or joined in. */
export const CONNECTION_ROLES = Object.keys(connectionRoles) as readonly ConnectionRole[]

/** The role of a link as the window names it; null for a value the schema leaves unspecified. */
export const readRole = (code: NoteRoleCode): LinkRole | null => calledRole[code]

/** The marker of a link as the window names it; null where it draws nothing. */
export const readMarker = (code: NoteMarkerCode): LinkMarker | null => calledMarker[code]

/** The marker of a link as the schema names it. */
export const writeMarker = (marker: LinkMarker | null | undefined): NoteMarkerCode =>
  marker ? markerCodes[marker] : NOTE_MARKER.UNSPECIFIED

/** The role of a connection as the window names it; null where it is not one a connection carries. */
const readConnectionRole = (code: NoteRoleCode): ConnectionRole | null => {
  const role = readRole(code)
  return role !== null && isConnectionRole(role) ? role : null
}

const isConnectionRole = (role: LinkRole): role is ConnectionRole => role in connectionRoles

interface SideWire {
  readonly label: string
  readonly note: string
  readonly marker: NoteMarkerCode
}

/** A connection as the schema answers with it. */
export interface ConnectionWire {
  readonly first: string
  readonly second: string
  readonly role: NoteRoleCode
  readonly firstSide?: SideWire | undefined
  readonly secondSide?: SideWire | undefined
}

const readSide = (side: SideWire | undefined): ConnectionSide | null =>
  side === undefined
    ? null
    : { label: side.label, marker: readMarker(side.marker), reason: side.note }

/** A connection in the words of the SDK. `type` is the type the answer was asked for. */
export const mapConnection = (wire: ConnectionWire, type: string): Connection => ({
  first: wire.first,
  second: wire.second,
  role: readConnectionRole(wire.role),
  type,
  firstSide: readSide(wire.firstSide),
  secondSide: readSide(wire.secondSide),
})

const writeSide = (side: ConnectionSide | null, type: string) =>
  side === null
    ? undefined
    : { type, note: side.reason, label: side.label, marker: writeMarker(side.marker) }

/** A connection in the shape the schema carries it. The roles of the records are left to the core. */
export const mapConnectionToWire = (connection: Connection) => ({
  first: connection.first,
  second: connection.second,
  role: connection.role === null ? NOTE_ROLE.UNSPECIFIED : roleCodes[connection.role],
  firstSide: writeSide(connection.firstSide, connection.type),
  secondSide: writeSide(connection.secondSide, connection.type),
})

/** The error an answer to a write of relationships names, and null where it landed. */
export const findRelationError = (answer: {
  readonly error?: number | undefined
  readonly isInvalid?: boolean | undefined
  readonly isCycle?: boolean | undefined
}): RelationError | null => {
  if (answer.isCycle === true) return 'cycle'
  if (answer.isInvalid === true) return 'invalid'
  return findErrorCode(answer)
}

interface BacklinkWire {
  readonly from?: { readonly path: string; readonly title: string } | undefined
  readonly role: NoteRoleCode
  readonly roleFromHere: NoteRoleCode
  readonly type: string
  readonly note: string
  readonly label: string
  readonly marker: NoteMarkerCode
}

/** The backlinks of an answer, in the words of the SDK. */
export const mapBacklinks = (backlinks: readonly BacklinkWire[]): Backlink[] =>
  backlinks.map((one) => ({
    from: { path: one.from?.path ?? '', title: one.from?.title ?? '' },
    role: readRole(one.role),
    roleFromHere: readRole(one.roleFromHere),
    type: one.type,
    reason: one.note,
    label: one.label,
    marker: readMarker(one.marker),
  }))
