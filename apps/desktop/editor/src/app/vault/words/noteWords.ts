/**
 * Note domain conversions between schema representations and window models.
 */
import {
  asFailure,
  asValue,
  findErrorCode,
  formatFingerprint,
  getNames,
  isStaleAnswer,
  NAME_SOURCE,
  NOTE_RELATION,
  NOTE_ROLE,
  parseFingerprint,
  type NameSourceCode,
  type NoteErrorCodeValue,
  type NoteRelationCode,
  type NoteRoleCode,
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

interface NeighbourhoodWire {
  readonly focus?: { readonly path: string; readonly title: string } | undefined
  readonly focusType: string
  readonly related: readonly {
    readonly note?: { readonly path: string; readonly title: string } | undefined
    readonly type: string
    readonly relation: NoteRelationCode
    readonly label: string
    readonly reverseLabel: string
    readonly through: string
    readonly isMutual: boolean
  }[]
}

/**
 * Whether a rename wrote the title into the note, by the namer it was named
 * with.
 */
export const writes: Record<NameSourceCode, boolean> = {
  [NAME_SOURCE.UNSPECIFIED]: false,
  [NAME_SOURCE.FRONTMATTER]: true,
  [NAME_SOURCE.FILENAME]: false,
}

/** What kind of relationship a link is, in the words the window uses. */
const called: Record<NoteRoleCode, Role | null> = {
  [NOTE_ROLE.UNSPECIFIED]: null,
  [NOTE_ROLE.PARENT]: 'parent',
  [NOTE_ROLE.CHILD]: 'child',
  [NOTE_ROLE.JUMP]: 'jump',
  [NOTE_ROLE.REF]: 'ref',
  [NOTE_ROLE.ATTACHMENT]: 'attachment',
}

/** What kind of relationship a link is, as the schema names it. */
export const roles = getNames<Role, NoteRoleCode>(called)

/** Where a note sits around the note in focus, in the words the window uses. */
export const relationByWire: Record<NoteRelationCode, NoteRelation | null> = {
  [NOTE_RELATION.UNSPECIFIED]: null,
  [NOTE_RELATION.PARENT]: 'parent',
  [NOTE_RELATION.CHILD]: 'child',
  [NOTE_RELATION.JUMP]: 'jump',
  [NOTE_RELATION.SIBLING]: 'sibling',
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
  error?: NoteErrorCodeValue | undefined
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
  error?: NoteErrorCodeValue | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  url?: string | undefined
  embed?: string | undefined
}): WriteResult => (isStaleAnswer(from) ? asFailure('changed') : mapNoteResult(from))

/**
 * A neighbourhood in the words the window uses.
 */
export const mapNeighbourhood = (answer: NeighbourhoodWire): Neighbourhood => ({
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
