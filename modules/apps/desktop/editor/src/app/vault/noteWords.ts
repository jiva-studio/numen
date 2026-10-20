/**
 * Note domain conversions between schema representations and window models.
 */
import {
  NameSource,
  NoteType as NoteTypes,
  Role as Roles,
  Relation as Relations,
} from '@numen/protocol'
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
} from '@numen/wire'
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

/** Which of four a note is, in the words the window uses. */
const typed: Record<NoteTypes, NoteType> = {
  [NoteTypes.UNSPECIFIED]: 'note',
  [NoteTypes.DECK]: 'deck',
  [NoteTypes.STENCIL]: 'stencil',
  [NoteTypes.PRESET]: 'preset',
}

export const mapNoteType = (of: NoteTypes): NoteType => typed[of] ?? 'note'

/** A run of text, kept as the plain pair the window carries it as. */
export const run = (span: { from: number; to: number }) => ({ from: span.from, to: span.to })

/** A link in the shape the schema carries it. */
export const mapLink = (link: Link) => ({
  to: link.to,
  role: roles[link.role],
  label: link.label ?? '',
})

export const mapBaseline = (baseline: { prose: string; fingerprint: string }) => ({
  prose: baseline.prose,
  at: parseFingerprint(baseline.fingerprint),
})

export const mapNoteResult = (from: {
  body?: string | undefined
  error?: ProtoErrorCode | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  url?: string | undefined
  embed?: string | undefined
}): NoteResult => {
  const error = findErrorCode(from)
  if (error) return asFailure(error)
  const at = formatFingerprint(from.at)
  const link = from.url === undefined ? undefined : { url: from.url, embed: from.embed ?? '' }
  return asValue({
    body: from.body ?? '',
    ...(at === undefined ? {} : { at }),
    ...(link === undefined ? {} : { link }),
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
