/**
 * A plex standing on a note, and the vault behind it, made up for a test.
 *
 * The vault is not here: what the view answers is written down when it is made,
 * and everything the editor was asked to do is kept in the order it was asked.
 */
import { ref } from 'vue'
import type { PlexEditor } from '../model/usePlexTab'
import type { PlexView } from '../model/usePlexView'
import type { Neighbourhood } from '@/entities/note'
import type { NoteType } from '@/entities/file'

/** A moment for whatever a gesture asked the vault for to come back. */
export const settle = () => new Promise((done) => setTimeout(done, 0))

/** Which of three each note of a neighbourhood is, by the path it stands at. */
export type Types = Record<string, NoteType>

/** A neighbourhood as the vault answers one: a focus, and what is around it. */
const createNeighbourhood = (
  focus: string,
  neighbours: readonly string[] = [],
  types: Types = {},
): Neighbourhood => ({
  focus: { path: focus, title: focus.replace(/\.md$/, '') },
  focusType: types[focus] ?? 'note',
  related: neighbours.map((path) => ({
    seat: 'child',
    through: '',
    label: '',
    isMutual: false,
    path,
    title: path.replace(/\.md$/, ''),
    type: types[path] ?? 'note',
  })),
})

/** A plex standing on a note, which records every note it was sent to. */
export const viewOn = (at: string, neighbours: readonly string[] = [], types: Types = {}) => {
  const went: string[] = []
  const view = {
    neighbourhood: ref(createNeighbourhood(at, neighbours, types)),
    here: ref(at),
    error: ref(''),
    go: async (path: string) => {
      went.push(path)
      view.here.value = path
      view.neighbourhood.value = createNeighbourhood(path, [], types)
    },
    followMoves: (renamed: readonly { from: string; to: string }[]) => {
      const one = renamed.find((went) => went.from === view.here.value)
      if (one) view.here.value = one.to
    },
    close: () => {},
  }
  return { view: view as unknown as PlexView, went }
}

/** A vault that takes every note it is asked to make, and records the asking. */
export const createVault = (takes = true) => {
  const made: [string, string][] = []
  const joined: [string, string, string][] = []
  /** The notes this vault will write no link to, which a test names. */
  const notJoinedPaths = new Set<string>()
  const editor: PlexEditor = {
    createInSeat: async (from, seat) => {
      made.push([from, seat])
      return takes ? { path: 'Made.md', title: 'Made' } : null
    },
    join: async (from, to, seat) => {
      joined.push([from, to, seat])
      return takes && !notJoinedPaths.has(to)
    },
  }
  return { editor, made, joined, notJoinedPaths }
}
