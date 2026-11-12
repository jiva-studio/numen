/**
 * A plex centered on a note, and the vault behind it, made up for a test.
 *
 * The vault is not here: what the view returns is recorded when it is created, and
 * every request the editor made is kept in order.
 */
import { ref } from 'vue'
import { asFailure, asValue, NOTE_TYPE, type Connection, type LinkMarker } from '@numen/plugin-sdk'
import type { PlexEditor, PlexView, NoteGraph, NoteType, PathRename, PlexRelation } from '../types'

/** A tick that lets whatever a gesture requested from the vault resolve. */
export const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

export type Types = Record<string, NoteType>

export interface RelatedNoteSpec {
  path: string
  relation?: PlexRelation
  through?: string
  label?: string
  reverseLabel?: string
  marker?: LinkMarker
  reverseMarker?: LinkMarker
  title?: string
  type?: NoteType
}

export type RelatedNoteInput = string | RelatedNoteSpec

const toSide = (label: string | undefined, marker: LinkMarker | undefined) =>
  label === undefined && marker === undefined
    ? null
    : { label: label ?? '', marker: marker ?? null, reason: '' }

/**
 * The line a related note is drawn by, as the vault answers it: from a parent to
 * its child, and from the related note to the focus for a jump.
 */
export const createLine = (focus: string, spec: RelatedNoteSpec): Connection => {
  const relation = spec.relation ?? 'child'
  const named = relation === 'sibling' ? (spec.through ?? '') : focus
  const ownSide = toSide(spec.label, spec.marker)
  const relatedSide = toSide(spec.reverseLabel, spec.reverseMarker)
  const isNamedFirst = relation === 'child' || relation === 'sibling'
  return {
    first: isNamedFirst ? named : spec.path,
    second: isNamedFirst ? spec.path : named,
    role: relation === 'jump' ? 'jump' : 'child',
    type: '',
    firstSide: isNamedFirst ? ownSide : relatedSide,
    secondSide: isNamedFirst ? relatedSide : ownSide,
  }
}

const createNoteGraph = (
  focus: string,
  relatedNotes: readonly RelatedNoteInput[] = [],
  types: Types = {},
  lines: readonly Connection[] = [],
): NoteGraph => {
  const specs = relatedNotes.map((item) => (typeof item === 'string' ? { path: item } : item))
  const focusKind = types[focus]
  const noteGraph: NoteGraph = {
    focus: { path: focus, title: focus.replace(/\.md$/, '') },
    focusType: focusKind ?? NOTE_TYPE,
    related: specs.map((spec) => ({
      relation: spec.relation ?? 'child',
      path: spec.path,
      title: spec.title ?? spec.path.replace(/\.md$/, ''),
      type: spec.type ?? types[spec.path] ?? NOTE_TYPE,
      ...(types[spec.path] ? { tabKind: types[spec.path] } : {}),
    })),
    connections: [...specs.map((spec) => createLine(focus, spec)), ...lines],
  }
  return focusKind ? { ...noteGraph, focusTabKind: focusKind } : noteGraph
}

/** A plex standing on a note, which records every note it was sent to. `lines` join notes other than the focus. */
export const viewOn = (
  initialPath: string,
  relatedNotes: readonly RelatedNoteInput[] = [],
  types: Types = {},
  lines: readonly Connection[] = [],
) => {
  const went: string[] = []
  const view: PlexView = {
    noteGraph: ref<NoteGraph | null>(createNoteGraph(initialPath, relatedNotes, types, lines)),
    here: ref(initialPath),
    error: ref(''),
    nextPath: ref(''),
    isLoading: ref(false),
    go: async (path: string) => {
      went.push(path)
      view.here.value = path
      view.noteGraph.value = createNoteGraph(
        path,
        path === initialPath ? relatedNotes : [],
        types,
        path === initialPath ? lines : [],
      )
    },
    followMoves: (renamed: readonly PathRename[]) => {
      const one = renamed.find((item) => item.from === view.here.value)
      if (one) view.here.value = one.to
    },
    close: () => undefined,
  }
  return { view, went }
}

/** A vault that accepts every note creation request, and records the requests. */
export const createVault = (takes = true, connection: Connection | null = null) => {
  const createdItem: [string, string][] = []
  const joined: [string, string, string, string?][] = []
  const saved: Connection[] = []
  const removed: [string, string][] = []
  /** The notes this vault will write no link to, which a test names. */
  const notJoinedPaths = new Set<string>()
  const editor: PlexEditor = {
    createRelated: async ({ from, role, title }) => {
      createdItem.push([from, role])
      return takes ? asValue(title ? `${title}.md` : 'Made.md') : asFailure('notFound')
    },
    join: async (from, to, side) => {
      if (side.label !== undefined) joined.push([from, to, side.role, side.label])
      else joined.push([from, to, side.role])
      return takes && !notJoinedPaths.has(to) ? asValue(undefined) : asFailure('notFound')
    },
    getConnection: async (first, second, type = '') =>
      asValue(
        connection ?? {
          first,
          second,
          role: null,
          type,
          firstSide: null,
          secondSide: null,
        },
      ),
    saveConnection: async (saving) => {
      saved.push(saving)
      return takes ? asValue(undefined) : asFailure('cycle')
    },
    removeConnection: async (first, second) => {
      removed.push([first, second])
      return takes ? asValue(undefined) : asFailure('notFound')
    },
  }
  return { editor, made: createdItem, joined, saved, removed, notJoinedPaths }
}
