/**
 * Which notes a node hangs parts for, asked without a screen.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import type { NoteHeading } from '@/entities/note'
import type { NoteType } from '@/entities/file'
import { createNodeIdMap } from '../lib/nodeIdMap'
import type { PlexTabDeps } from '../types'
import { usePlexParts } from './usePlexParts'

/** A window that hangs parts and answers with the headings it is given. */
const createDeps = (
  headings: ReadonlyMap<string, readonly NoteHeading[]>,
  asked: (readonly string[])[],
): PlexTabDeps => ({
  editor: {
    createInSeat: async () => null,
    createWithTitle: async () => null,
    join: async () => false,
    removeLink: async () => false,
  },
  isReady: ref(true),
  isHanging: ref(true),
  parts: ref(6),
  openNote: () => {},
  readHeadings: async (paths) => {
    asked.push(paths)
    return headings
  },
  askAgent: () => {},
  runCommand: () => {},
  openingPath: ref(''),
  dragged: ref([]),
  showMessage: () => {},
  createUntitledNote: async () => '',
  readOpeningPath: async () => '',
  creatable: [],
})

describe('the notes whose parts are read', () => {
  it('include a path the types say nothing about', async () => {
    const asked: (readonly string[])[] = []
    const heading: NoteHeading = { line: 3, text: 'Intro', level: 1 }
    const map = createNodeIdMap()
    const node = map.getNodeId('Loose.md')
    const plex = usePlexParts(
      ref(['Loose.md']),
      ref<ReadonlyMap<string, NoteType>>(new Map()),
      createDeps(new Map([['Loose.md', [heading]]]), asked),
      map,
    )

    await plex.readParts()

    expect(asked.at(-1)).toStrictEqual(['Loose.md'])
    expect(plex.getParts(node)).toStrictEqual([{ id: '3', text: 'Intro', level: 1 }])
  })
})
