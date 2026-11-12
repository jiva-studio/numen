/**
 * Relationship conversions between schema representations and the SDK models.
 */
import {
  NOTE_MARKER,
  NOTE_ROLE,
  findErrorCode,
  getNames,
  type NoteMarkerCode,
  type NoteRoleCode,
} from '@numen/core-client'
import type {
  Backlink,
  Connection,
  ConnectionRole,
  ConnectionSide,
  LinkMarker,
  LinkRole,
  RelationError,
} from '@numen/plugin-loader'

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

const readRole = (code: NoteRoleCode): LinkRole | null => calledRole[code]

const readMarker = (code: NoteMarkerCode): LinkMarker | null => calledMarker[code]

/** The marker of a link as the schema names it. */
export const writeMarker = (marker: LinkMarker | null | undefined): NoteMarkerCode =>
  marker ? markerCodes[marker] : NOTE_MARKER.UNSPECIFIED

const readConnectionRole = (code: NoteRoleCode): ConnectionRole | null => {
  const role = readRole(code)
  return isConnectionRole(role) ? role : null
}

const isConnectionRole = (role: LinkRole | null): role is ConnectionRole =>
  // Stryker disable next-line ConditionalExpression: 'null' is not a key in connectionRoles
  role !== null && role in connectionRoles

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
  readonly sourceRole: NoteRoleCode
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
    sourceRole: readRole(one.sourceRole),
    type: one.type,
    reason: one.note,
    label: one.label,
    marker: readMarker(one.marker),
  }))
