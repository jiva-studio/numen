/**
 * Note domain conversions between schema representations and window models.
 */
import { NameSource, Role as Roles, Relation as Relations } from '@numen/protocol'
import type {
  ErrorCode as ProtoErrorCode,
  GetNeighbourhoodResponse as NeighbourhoodMessage,
} from '@numen/protocol'
import {
  asFailure,
  asValue,
  findErrorCode,
  formatFingerprint,
  getNames,
  isStaleAnswer,
  parseFingerprint,
} from '@numen/plugin-sdk'
import type {
  Link,
  Neighbourhood,
  NoteResult,
  NoteRelation,
  Role,
  WriteResult,
} from '@/entities/note'
import type { NoteType } from '@/entities/file'

/**
 * Whether a rename wrote the title into the note, by the namer it was named
 * with.
 */
export const writes: Record<NameSource, boolean> = {
  [NameSource.UNSPECIFIED]: false,
  [NameSource.FRONTMATTER]: true,
  [NameSource.FILENAME]: false,
}

/** What kind of relationship a link is, in the words the window uses. */
const called: Record<Roles, Role | null> = {
  [Roles.UNSPECIFIED]: null,
  [Roles.PARENT]: 'parent',
  [Roles.CHILD]: 'child',
  [Roles.JUMP]: 'jump',
  [Roles.REF]: 'ref',
  [Roles.ATTACHMENT]: 'attachment',
}

/** What kind of relationship a link is, as the schema names it. */
export const roles = getNames<Role, Roles>(called)

/** Where a note sits around the note in focus, in the words the window uses. */
export const relationByWire: Record<Relations, NoteRelation | null> = {
  [Relations.UNSPECIFIED]: null,
  [Relations.PARENT]: 'parent',
  [Relations.CHILD]: 'child',
  [Relations.JUMP]: 'jump',
  [Relations.SIBLING]: 'sibling',
}

/** What a note is, where the schema leaves it empty for a plain note. */
export const mapNoteType = (of: string): NoteType => of || 'note'

/** Copies a wire range into the plain pair the window carries. */
export const mapRange = (range: { start: number; end: number }) => ({
  start: range.start,
  end: range.end,
})

/** A link in the shape the schema carries it. */
export const mapLink = (link: Link) => ({
  to: link.to,
  role: roles[link.role],
  label: link.label ?? '',
})

export const mapBaseline = (baseline: { prose: string; stat: string }) => ({
  prose: baseline.prose,
  at: parseFingerprint(baseline.stat),
})

export const mapNoteResult = (from: {
  body?: string | undefined
  error?: ProtoErrorCode | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
}): NoteResult => {
  const error = findErrorCode(from)
  if (error) return asFailure(error)
  const stat = formatFingerprint(from.at)
  return asValue({
    body: from.body ?? '',
    ...(stat === undefined ? {} : { at: stat }),
  })
}

/** What a write came back with. A file that moved past the caller has changed. */
export const mapWriteResult = (from: {
  body?: string | undefined
  error?: ProtoErrorCode | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  url?: string | undefined
  embed?: string | undefined
}): WriteResult => (isStaleAnswer(from) ? asFailure('changed') : mapNoteResult(from))

/**
 * A neighbourhood in the words the window uses.
 */
export const mapNeighbourhood = (answer: NeighbourhoodMessage): Neighbourhood => ({
  focus: { path: answer.focus?.path ?? '', title: answer.focus?.title ?? '' },
  focusType: mapNoteType(answer.focusType),
  related: answer.related.flatMap((one) => {
    const relation = relationByWire[one.relation] ?? null
    if (relation === null || one.note === undefined) return []
    return [
      {
        path: one.note.path,
        title: one.note.title,
        type: mapNoteType(one.type),
        relation,
        label: one.label,
        reverseLabel: one.reverseLabel,
        through: one.through,
        isMutual: one.isMutual,
      },
    ]
  }),
})
