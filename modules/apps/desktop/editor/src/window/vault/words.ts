/**
 * Conversions between schema representations and window domain models.
 */
import {
  BookFormat as BookFormats,
  FlushResult,
  NamedBy,
  NoteType as NoteTypes,
  Presence as Presences,
  Role as Roles,
  SearchMode as Modes,
  Seat as Seats,
  SourceKind,
  Unit as Units,
  VaultsRefusal,
} from '@numen/protocol'
import type {
  Entry as EntryMessage,
  GetNeighbourhoodResponse as NeighbourhoodMessage,
  MoveResult as MoveResultMessage,
  Refusal,
  Vault as VaultMessage,
} from '@numen/protocol'
import type { TallyUnit } from '@numen/ui'
import { namesOf } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '../../shared/answers'
import type { SearchMode } from '../../shared/command/search'
import type {
  BookFormat,
  Entry,
  Link,
  MoveResult,
  Neighbourhood,
  NoteResult,
  NoteType,
  Presence,
  Role,
  Seat,
  Source,
  Vault,
  VaultRefusalReason,
  VaultResult,
} from '../../shared/core'

/**
 * What a piece of work counts, in the words the window uses. One it has no word
 * for is counted one by one.
 */
export const counted: Record<Units, TallyUnit> = {
  [Units.UNSPECIFIED]: 'things',
  [Units.THINGS]: 'things',
  [Units.BYTES]: 'bytes',
  [Units.SECONDS]: 'seconds',
}

/**
 * Whether a rename wrote the title into the note, by the namer it was named
 * with. Keyed by the schema, so a namer added to it is answered here before
 * this compiles.
 */
export const writes: Record<NamedBy, boolean> = {
  [NamedBy.UNSPECIFIED]: false,
  [NamedBy.FRONTMATTER]: true,
  [NamedBy.FILENAME]: false,
}

/**
 * How a search is asked, in the words the window uses. Keyed by the schema, so
 * a mode added to it has to be given a word here before this compiles.
 */
const asked: Record<Modes, SearchMode | null> = {
  [Modes.UNSPECIFIED]: null,
  [Modes.HYBRID]: 'hybrid',
  [Modes.WORDS]: 'words',
  [Modes.MEANING]: 'meaning',
  [Modes.NAMES]: 'names',
}

/** How a search is asked, as the schema names it. */
export const modes = namesOf<SearchMode, Modes>(asked)

/**
 * What kind of relationship a link is, in the words the window uses. Keyed by
 * the schema, so a role added to it has to be given a word here before this
 * compiles, and the window cannot quietly go on knowing four of five.
 */
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

/** A source this window has no word for is a file it holds no source for. */
const holding: Record<SourceKind, Source> = {
  [SourceKind.UNSPECIFIED]: 'other',
  [SourceKind.NOTE]: 'note',
  [SourceKind.BOOK]: 'book',
  [SourceKind.RECORDING]: 'recording',
  [SourceKind.URL]: 'url',
}

export const sourceKind = (of: SourceKind): Source => holding[of] ?? 'other'

/** How a book at a path is drawn, in the words the window uses. */
const drawn: Record<BookFormats, BookFormat | undefined> = {
  [BookFormats.UNSPECIFIED]: undefined,
  [BookFormats.PDF]: 'pdf',
  [BookFormats.EPUB]: 'epub',
}

export const bookFormat = (of: BookFormats): BookFormat | undefined => drawn[of]

/** What a model's files are on this machine, in the words the window uses. */
export const fetched: Record<Presences, Presence> = {
  [Presences.UNSPECIFIED]: 'nothing to fetch',
  [Presences.PRESENT]: 'present',
  [Presences.NOT_FETCHED]: 'not fetched',
  [Presences.NOTHING_TO_FETCH]: 'nothing to fetch',
}

/** Where a note sits around the note in focus, in the words the window uses. */
const seated: Record<Seats, Seat | null> = {
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

const unvaulted: Record<VaultsRefusal, VaultRefusalReason> = {
  [VaultsRefusal.UNSPECIFIED]: 'unreadable',
  [VaultsRefusal.UNREADABLE]: 'unreadable',
  [VaultsRefusal.COPY]: 'copy',
  [VaultsRefusal.OVERLAPS]: 'overlaps',
  [VaultsRefusal.NAME_TAKEN]: 'nameTaken',
  [VaultsRefusal.LAST_VAULT]: 'lastVault',
  [VaultsRefusal.SHOWING]: 'showing',
  [VaultsRefusal.UNKNOWN]: 'unknown',
  [VaultsRefusal.NO_TRASH]: 'noTrash',
  [VaultsRefusal.ASKING]: 'asking',
}

export const turnedDown = (from: { refusal?: VaultsRefusal | undefined }): VaultRefusalReason | null =>
  from.refusal === undefined ? null : unvaulted[from.refusal]

/**
 * What a client has left, in the words the window uses. Keyed by the schema,
 * so a result added to it has to be given a word here before this compiles.
 */
const left: Record<FlushResult, 'nothing' | 'written' | 'asking' | null> = {
  [FlushResult.UNSPECIFIED]: null,
  [FlushResult.NOTHING]: 'nothing',
  [FlushResult.WRITTEN]: 'written',
  [FlushResult.ASKING]: 'asking',
}

export const owing = namesOf<NonNullable<(typeof left)[FlushResult]>, FlushResult>(left)

/** A run of text, kept as the plain pair the window carries it as. */
export const run = (span: { from: number; to: number }) => ({ from: span.from, to: span.to })

/** A link in the shape the schema carries it. */
export const written = (link: Link) => ({
  to: link.to,
  role: roles[link.role],
  label: link.label ?? '',
})

export const seenOf = (seen: { prose: string; path: string }) => ({
  prose: seen.prose,
  at: fingerprint(seen.path),
})

export const answered = (from: {
  body?: string | undefined
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
    refusal: error,
    changed: staleIn(from),
    ...(at === undefined ? {} : { at }),
    ...(link === undefined ? {} : { link, address: link }),
  }
}

/** One row of a listing, kept as the plain value the window carries it as. */
export const listed = (one: EntryMessage): Entry => ({
  path: one.path,
  name: one.name,
  folder: one.folder,
  kind: sourceKind(one.kind),
  type: noteType(one.type),
})

/**
 * A neighbourhood in the words the window uses. A note the window has no seat
 * for, and one the answer names no note at, is not one of them.
 */
export const around = (said: NeighbourhoodMessage): Neighbourhood => ({
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

/** One vault of the list, kept as the plain value the window carries it as. */
export const held = (one: VaultMessage): Vault => ({
  id: one.id,
  name: one.name,
  path: one.path,
  missing: one.missing,
})

export const added = (from: {
  vault?: VaultMessage | undefined
  refusal?: VaultsRefusal | undefined
}): VaultResult => {
  const error = turnedDown(from)
  return {
    vault: from.vault ? held(from.vault) : null,
    error,
    refusal: error,
  }
}

/** What the file did, in the shape the window carries it. */
export const filed = (moved: MoveResultMessage): MoveResult => ({
  from: moved.from,
  to: moved.to,
  repaired: moved.repaired,
})
