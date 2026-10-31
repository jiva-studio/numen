/**
 * A plex centred on a note, and the vault behind it, made up for a test.
 *
 * The vault is not here: what the view returns is recorded when it is created, and
 * every request the editor made is kept in order.
 */
import { ref } from 'vue'
import { asFailure, asValue, NOTE_TYPE, type Connection, type LinkMarker } from '@numen/plugin-sdk'
import type { PlexRelation } from '../plex'
import type { PlexEditor, PlexView, Neighbourhood, NoteType, PathRename } from '../types'

/** A tick that lets whatever a gesture requested from the vault resolve. */
export const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

export type Types = Record<string, NoteType>

export interface RelatedNeighbourSpec {
  path: string
  relation?: PlexRelation
  through?: string
  label?: string
  reverseLabel?: string
  isMutual?: boolean
  marker?: LinkMarker
  reverseMarker?: LinkMarker
  title?: string
  type?: NoteType
}

export type NeighbourSpec = string | RelatedNeighbourSpec

/** A neighbourhood as the vault answers one: a focus, and what is around it. */
const createNeighbourhood = (
  focus: string,
  neighbours: readonly NeighbourSpec[] = [],
  types: Types = {},
): Neighbourhood => ({
  focus: { path: focus, title: focus.replace(/\.md$/, '') },
  focusType: types[focus] ?? NOTE_TYPE,
  related: neighbours.map((item) => {
    const spec: RelatedNeighbourSpec = typeof item === 'string' ? { path: item } : item
    return {
      relation: spec.relation ?? 'child',
      through: spec.through ?? '',
      label: spec.label ?? '',
      reverseLabel: spec.reverseLabel ?? '',
      isMutual: spec.isMutual ?? false,
      marker: spec.marker ?? null,
      reverseMarker: spec.reverseMarker ?? null,
      path: spec.path,
      title: spec.title ?? spec.path.replace(/\.md$/, ''),
      type: spec.type ?? types[spec.path] ?? NOTE_TYPE,
    }
  }),
})

/** A plex standing on a note, which records every note it was sent to. */
export const viewOn = (
  initialPath: string,
  neighbours: readonly NeighbourSpec[] = [],
  types: Types = {},
) => {
  const went: string[] = []
  const view: PlexView = {
    neighbourhood: ref<Neighbourhood | null>(createNeighbourhood(initialPath, neighbours, types)),
    here: ref(initialPath),
    error: ref(''),
    nextPath: ref(''),
    isLoading: ref(false),
    go: async (path: string) => {
      went.push(path)
      view.here.value = path
      view.neighbourhood.value = createNeighbourhood(
        path,
        path === initialPath ? neighbours : [],
        types,
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
      return takes ? asValue(title ? `${title}.md` : 'Made.md') : asFailure('missing')
    },
    join: async (from, to, side) => {
      if (side.label !== undefined) joined.push([from, to, side.role, side.label])
      else joined.push([from, to, side.role])
      return takes && !notJoinedPaths.has(to) ? asValue(undefined) : asFailure('missing')
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
      return takes ? asValue(undefined) : asFailure('missing')
    },
  }
  return { editor, made: createdItem, joined, saved, removed, notJoinedPaths }
}
