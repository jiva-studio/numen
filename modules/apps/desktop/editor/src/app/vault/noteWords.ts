/**
 * Note domain conversions between schema representations and window models.
 */
import {
  NamedBy,
  NoteType as NoteTypes,
  Role as Roles,
  Seat as Seats,
} from '@numen/protocol'
import type {
  GetNeighbourhoodResponse as NeighbourhoodMessage,
  Refusal,
} from '@numen/protocol'
import { namesOf } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '@/shared/answers'
import type {
  Link,
  Neighbourhood,
  NoteResult,
  NoteType,
  Role,
  Seat,
} from '@/shared/core'

/**
 * Whether a rename wrote the title into the note, by the namer it was named
 * with.
 */
export const writes: Record<NamedBy, boolean> = {
  [NamedBy.UNSPECIFIED]: false,
  [NamedBy.FRONTMATTER]: true,
  [NamedBy.FILENAME]: false,
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
export const roles = namesOf<Role, Roles>(called)

/** Where a note sits around the note in focus, in the words the window uses. */
export const seated: Record<Seats, Seat | null> = {
  [Seats.UNSPECIFIED]: null,
  [Seats.PARENT]: 'parent',
  [Seats.CHILD]: 'child',
  [Seats.JUMP]: 'jump',
  [Seats.SIBLING]: 'sibling',
}

/** Which of four a note is, in the words the window uses. */
const typed: Record<NoteTypes, NoteType> = {
  [NoteTypes.UNSPECIFIED]: 'note',
  [NoteTypes.DECK]: 'deck',
  [NoteTypes.STENCIL]: 'stencil',
  [NoteTypes.PRESET]: 'preset',
}

export const noteType = (of: NoteTypes): NoteType => typed[of] ?? 'note'

/** A run of text, kept as the plain pair the window carries it as. */
export const run = (span: { from: number; to: number }) => ({ from: span.from, to: span.to })

/** A link in the shape the schema carries it. */
export const mapLink = (link: Link) => ({
  to: link.to,
  role: roles[link.role],
  label: link.label ?? '',
})

export const mapBaseline = (seen: { prose: string; at: string }) => ({
  prose: seen.prose,
  at: fingerprint(seen.at),
})

export const mapNoteResult = (from: {
  body?: string | undefined
  error?: Refusal | undefined
  refusal?: Refusal | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  url?: string | undefined
  embed?: string | undefined
}): NoteResult & { at?: string; changed: boolean } => {
  const at = stamp(from.at)
  const error = errorIn(from)
  const link = from.url === undefined ? undefined : { url: from.url, embed: from.embed ?? '' }
  return {
    body: from.body ?? '',
    error,
    changed: staleIn(from),
    ...(at === undefined ? {} : { at }),
    ...(link === undefined ? {} : { link }),
  }
}

/**
 * A neighbourhood in the words the window uses.
 */
export const mapNeighbourhood = (said: NeighbourhoodMessage): Neighbourhood => ({
  focus: { path: said.focus?.path ?? '', title: said.focus?.title ?? '' },
  focusType: noteType(said.focusType),
  related: said.related.flatMap((one) => {
    const seat = seated[one.seat] ?? null
    if (seat === null || one.note === undefined) return []
    return [
      {
        path: one.note.path,
        title: one.note.title,
        type: noteType(one.type),
        seat,
        label: one.label,
        through: one.through,
        isMutual: one.mutual,
        mutual: one.mutual,
      },
    ]
  }),
})
