/**
 * Note domain conversions between schema representations and window models.
 */
import { formatFingerprint } from '@numen/plugin-loader'
import {
  findErrorCode,
  isStaleAnswer,
  NAME_SOURCE,
  NOTE_LINK_DIRECTION,
  NOTE_RELATION,
  parseFingerprint,
  type NameSourceCode,
  type NoteErrorCodeValue,
  type NoteLinkDirectionCode,
  type NoteRelationCode,
  asFailure,
  asValue,
} from '@numen/core-client'
import type { LinkDirection, NoteGraph, NoteResult, NoteRelation, WriteResult } from '../lib/note'
import { mapNoteType } from '@/shared/api/wire'
import { mapConnection, type ConnectionWire } from './relationWords'

interface NoteGraphWire {
  readonly focus?: { readonly path: string; readonly title: string } | undefined
  readonly focusType: string
  readonly focusTabKind?: string
  readonly related: readonly {
    readonly note?: { readonly path: string; readonly title: string } | undefined
    readonly type: string
    readonly tabKind?: string
    readonly relation: NoteRelationCode
    readonly direction?: NoteLinkDirectionCode
    readonly relatedTo?: string
  }[]
  readonly connections: readonly ConnectionWire[]
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

/** The position of a note relative to the focused note, as named in the window. */
const relationByWire: Record<NoteRelationCode, NoteRelation | null> = {
  [NOTE_RELATION.UNSPECIFIED]: null,
  [NOTE_RELATION.PARENT]: 'parent',
  [NOTE_RELATION.CHILD]: 'child',
  [NOTE_RELATION.JUMP]: 'jump',
  [NOTE_RELATION.SIBLING]: 'sibling',
  [NOTE_RELATION.REFERENCE]: 'reference',
}

/** Which note wrote a text link, as named in the window. An unspecified one has no direction. */
const directionByWire: Record<NoteLinkDirectionCode, LinkDirection | null> = {
  [NOTE_LINK_DIRECTION.UNSPECIFIED]: null,
  [NOTE_LINK_DIRECTION.OUTGOING]: 'outgoing',
  [NOTE_LINK_DIRECTION.INCOMING]: 'incoming',
  [NOTE_LINK_DIRECTION.BOTH]: 'both',
}

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

/** What a write returned. A file that moved past the caller has changed. */
export const mapWriteResult = (from: {
  body?: string | undefined
  error?: NoteErrorCodeValue | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  url?: string | undefined
  embed?: string | undefined
}): WriteResult => (isStaleAnswer(from) ? asFailure('changed') : mapNoteResult(from))

/**
 * A note graph in the words the window uses.
 */
export const mapNoteGraph = (answer: NoteGraphWire): NoteGraph => ({
  focus: { path: answer.focus?.path ?? '', title: answer.focus?.title ?? '' },
  focusType: mapNoteType(answer.focusType),
  ...(answer.focusTabKind ? { focusTabKind: answer.focusTabKind } : {}),
  related: answer.related.flatMap((one) => {
    const relation = relationByWire[one.relation] ?? null
    if (relation === null || one.note === undefined) return []
    const direction =
      relation === 'reference' && one.direction !== undefined
        ? (directionByWire[one.direction] ?? null)
        : null
    return [
      {
        path: one.note.path,
        title: one.note.title,
        type: mapNoteType(one.type),
        ...(one.tabKind ? { tabKind: one.tabKind } : {}),
        relation,
        ...(direction === null ? {} : { direction }),
        ...(one.relatedTo ? { relatedTo: one.relatedTo } : {}),
      },
    ]
  }),
  connections: answer.connections.map((one) => mapConnection(one, '')),
})
