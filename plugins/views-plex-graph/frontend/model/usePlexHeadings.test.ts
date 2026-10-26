/**
 * Which notes a node hangs headings for, asked without a screen.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { createNodeIdMap } from '../lib/nodeIdMap'
import type { PlexTabDeps, NoteHeading, NoteType } from '../types'
import { usePlexHeadings } from './usePlexHeadings'

/** A window that hangs headings and answers with the headings it is given. */
const createDeps = (
  headings: ReadonlyMap<string, readonly NoteHeading[]>,
  request: (readonly string[])[],
  readHeadings: PlexTabDeps['readHeadings'] = async (paths) => {
    request.push(paths)
    return headings
  },
): PlexTabDeps => ({
  editor: {
    createInRelation: async () => null,
    createWithTitle: async () => null,
    join: async () => false,
    replaceLinks: async () => false,
  },
  isReady: ref(true),
  isNesting: ref(true),
  headings: ref(6),
  openNote: () => {},
  readHeadings,
  getNoteCommands: () => [],
  getCommandIcon: () => null,
  getNoteIcon: () => null,
  runCommand: () => {},
  openingPath: ref(''),
  dragged: ref([]),
  showMessage: () => {},
  createUntitledNote: async () => '',
  readOpeningPath: async () => '',
  creatable: [],
})

describe('the notes whose headings are read', () => {
  it('include a path the types say nothing about', async () => {
    const request: (readonly string[])[] = []
    const heading: NoteHeading = { line: 3, text: 'Intro', level: 1 }
    const map = createNodeIdMap()
    const node = map.getNodeId('Loose.md')
    const plex = usePlexHeadings(
      ref(['Loose.md']),
      ref<ReadonlyMap<string, NoteType>>(new Map()),
      createDeps(new Map([['Loose.md', [heading]]]), request),
      map,
    )

    await plex.readHeadings()

    expect(request.at(-1)).toStrictEqual(['Loose.md'])
    expect(plex.getHeadings(node)).toStrictEqual([{ id: '3', text: 'Intro', level: 1 }])
  })
})

describe('the headings of a node', () => {
  const heading: NoteHeading = { line: 3, text: 'Intro', level: 1 }
  const types = () => ref<ReadonlyMap<string, NoteType>>(new Map([['A.md', 'note']]))

  it('are none for a node nothing is drawn for', async () => {
    const map = createNodeIdMap()
    const plex = usePlexHeadings(
      ref(['A.md']),
      types(),
      createDeps(new Map([['A.md', [heading]]]), []),
      map,
    )

    await plex.readHeadings()

    expect(plex.getHeadings('missing')).toStrictEqual([])
  })

  it('are dropped where the vault cannot be read', async () => {
    const map = createNodeIdMap()
    const node = map.getNodeId('A.md')
    let isFailing = false
    const plex = usePlexHeadings(
      ref(['A.md']),
      types(),
      createDeps(new Map([['A.md', [heading]]]), [], async () => {
        if (isFailing) throw new Error('unreadable')
        return new Map([['A.md', [heading]]])
      }),
      map,
    )
    await plex.readHeadings()
    expect(plex.getHeadings(node)).toHaveLength(1)

    isFailing = true
    await plex.readHeadings()

    expect(plex.getHeadings(node)).toStrictEqual([])
  })

  it('are left alone by a read that failed after a newer one was requested', async () => {
    const map = createNodeIdMap()
    const node = map.getNodeId('A.md')
    let fail = () => {}
    let calls = 0
    const plex = usePlexHeadings(
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
    await plex.readHeadings()
    fail()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(plex.getHeadings(node)).toStrictEqual([{ id: '3', text: 'Intro', level: 1 }])
  })
})
