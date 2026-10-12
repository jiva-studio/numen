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
  readHeadings: PlexTabDeps['readHeadings'] = async (paths) => {
    asked.push(paths)
    return headings
  },
): PlexTabDeps => ({
  editor: {
    createInSeat: async () => null,
    createWithTitle: async () => null,
    join: async () => false,
    removeLink: async () => false,
  },
  isReady: ref(true),
  isNesting: ref(true),
  parts: ref(6),
  openNote: () => {},
  readHeadings,
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

describe('the parts of a node', () => {
  const heading: NoteHeading = { line: 3, text: 'Intro', level: 1 }
  const types = () => ref<ReadonlyMap<string, NoteType>>(new Map([['A.md', 'note']]))

  it('are none for a node nothing is drawn for', async () => {
    const map = createNodeIdMap()
    const plex = usePlexParts(
      ref(['A.md']),
      types(),
      createDeps(new Map([['A.md', [heading]]]), []),
      map,
    )

    await plex.readParts()

    expect(plex.getParts('missing')).toStrictEqual([])
  })

  it('are dropped where the vault cannot be read', async () => {
    const map = createNodeIdMap()
    const node = map.getNodeId('A.md')
    let isFailing = false
    const plex = usePlexParts(
      ref(['A.md']),
      types(),
      createDeps(new Map([['A.md', [heading]]]), [], async () => {
        if (isFailing) throw new Error('unreadable')
        return new Map([['A.md', [heading]]])
      }),
      map,
    )
    await plex.readParts()
    expect(plex.getParts(node)).toHaveLength(1)

    isFailing = true
    await plex.readParts()

    expect(plex.getParts(node)).toStrictEqual([])
  })

  it('are left alone by a read that failed after a newer one was asked', async () => {
    const map = createNodeIdMap()
    const node = map.getNodeId('A.md')
    let fail = () => {}
    let calls = 0
    const plex = usePlexParts(
      ref(['A.md']),
      types(),
      createDeps(new Map(), [], async () => {
        calls += 1
        if (calls > 1) return new Map([['A.md', [heading]]])
        await new Promise<void>((_, reject) => (fail = () => reject(new Error('late'))))
        return new Map()
      }),
      map,
    )
    // The first read is the one the watcher asks for on creation, and stays held.
    await plex.readParts()
    fail()
    await new Promise((wake) => setTimeout(wake, 0))

    expect(plex.getParts(node)).toStrictEqual([{ id: '3', text: 'Intro', level: 1 }])
  })
})
