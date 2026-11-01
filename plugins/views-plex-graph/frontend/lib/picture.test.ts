import { describe, expect, it } from 'vitest'
import type { Connection, ConnectionSide, LinkMarker } from '@numen/plugin-sdk'
import type { Neighbourhood, NoteRelation } from '../types'
import { areNeighbourhoodsEqual, convertNeighbourhood, getNoteTypes } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/** A related note: where it is, and how it sits relative to the focus. */
type NeighbourRow = [path: string, relation: NoteRelation]

/** A record one note writes about the other: its label and what it draws at the other note. */
type Stated = readonly [label: string, marker: LinkMarker | null]

const toSide = (record: Stated | null): ConnectionSide | null =>
  record === null ? null : { label: record[0], marker: record[1], reason: '' }

/** The connection between two notes, running from the first to the second. */
const createLine = (
  first: string,
  second: string,
  firstRecord: Stated | null = null,
  secondRecord: Stated | null = null,
): Connection => ({
  first,
  second,
  role: 'child',
  type: '',
  firstSide: toSide(firstRecord),
  secondSide: toSide(secondRecord),
})

const createNeighbourhood = (
  focus: string,
  rows: NeighbourRow[],
  lines: Connection[] = [],
): Neighbourhood => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, relation]) => ({ path, title: path, type: 'note', relation })),
  connections: lines,
})

/**
 * The graph, and the note each node ID in it identifies. Every note here has no
 * identifier, which is the case the graph has to render.
 */
const renderImage = (neighbourhood: Neighbourhood) => {
  const map = createNodeIdMap()
  return { plex: convertNeighbourhood(neighbourhood, map.getNodeId), note: map.getNodePath }
}

/**
 * Every edge, as the notes at its two ends. A token that no note holds resolves
 * to nothing, so an edge built from any other token is visible here.
 */
const getEdges = (neighbourhood: Neighbourhood) => {
  const { plex, note } = renderImage(neighbourhood)
  return plex.edges.map(
    (edge) =>
      `${note(edge.from)} -> ${note(edge.to)}${edge.labels ? ` (${edge.labels.map((label) => label.text).join(', ')})` : ''}`,
  )
}

describe('what the plex is handed', () => {
  it('runs an edge the way the connection runs', () => {
    expect(
      getEdges(
        createNeighbourhood(
          'Here',
          [
            ['Above', 'parent'],
            ['Below', 'child'],
            ['Across', 'jump'],
          ],
          [
            createLine('Above', 'Here', null, ['part of', null]),
            createLine('Here', 'Below'),
            createLine('Across', 'Here', ['see also', null]),
          ],
        ),
      ),
    ).toEqual(['Above -> Here (part of)', 'Here -> Below', 'Across -> Here (see also)'])
  })

  it('writes nothing on a line the person wrote nothing on', () => {
    // A line carries only the label the user put on it.
    const [line] = renderImage(
      createNeighbourhood('Here', [['Below', 'child']], [createLine('Here', 'Below', ['', null])]),
    ).plex.edges
    expect(line?.labels).toBeUndefined()
  })

  it('draws a line from the connection that joins its notes, whichever notes they are', () => {
    expect(
      getEdges(
        createNeighbourhood(
          'Here',
          [
            ['Above', 'parent'],
            ['Beside', 'sibling'],
          ],
          [createLine('Above', 'Here'), createLine('Above', 'Beside')],
        ),
      ),
    ).toEqual(['Above -> Here', 'Above -> Beside'])
  })

  it('draws no line for a connection to a note the picture does not show', () => {
    // The note a connection names that this picture never drew is not a node.
    const request: string[] = []
    const plex = convertNeighbourhood(
      createNeighbourhood('Here', [['Beside', 'sibling']], [createLine('Missing', 'Beside')]),
      (path) => {
        request.push(path)
        return `#${path}`
      },
    )

    expect(plex.edges).toEqual([])
    expect(request).not.toContain('Missing')
  })

  it('places every note it was given', () => {
    const { plex, note } = renderImage(
      createNeighbourhood('Here', [
        ['Above', 'parent'],
        ['Below', 'child'],
        ['Beside', 'sibling'],
      ]),
    )
    expect(plex.nodes.map((node) => `${node.relation} ${note(node.id)}`)).toEqual([
      'focus Here',
      'parent Above',
      'child Below',
      'sibling Beside',
    ])
  })
})

describe('what a note is drawn under', () => {
  const target = createNeighbourhood(
    'Here',
    [
      ['Above', 'parent'],
      ['Below', 'child'],
      ['Beside', 'sibling'],
    ],
    [createLine('Above', 'Here'), createLine('Here', 'Below'), createLine('Above', 'Beside')],
  )

  it('is the token the note holds, and never the file it is filed under', () => {
    const map = createNodeIdMap()
    const plex = convertNeighbourhood(target, (path) => `#${path}`)

    expect(plex.nodes.map((node) => node.id)).toEqual(['#Here', '#Above', '#Below', '#Beside'])
    expect(map.getNodePath('Here')).toBe('')
  })

  it('calls every note of one picture something different from every other', () => {
    const { plex } = renderImage(target)

    expect(new Set(plex.nodes.map((node) => node.id)).size).toBe(plex.nodes.length)
  })

  it('runs an edge between the tickets of the two notes it joins', () => {
    const plex = convertNeighbourhood(target, (path) => `#${path}`)

    expect(plex.edges.map((edge) => `${edge.from} -> ${edge.to}`)).toEqual([
      '#Above -> #Here',
      '#Here -> #Below',
      '#Above -> #Beside',
    ])
  })
})

/**
 * Every kind of line the plex draws, as a picture holding it. `first` is the
 * note the connection starts at and `second` the one it ends at.
 */
const LINE_KINDS: readonly {
  readonly name: string
  readonly rows: NeighbourRow[]
  readonly first: string
  readonly second: string
}[] = [
  { name: 'centre to parent', rows: [['Above', 'parent']], first: 'Above', second: 'Here' },
  { name: 'centre to child', rows: [['Below', 'child']], first: 'Here', second: 'Below' },
  { name: 'centre to jump', rows: [['Across', 'jump']], first: 'Across', second: 'Here' },
  {
    name: 'parent to sibling',
    rows: [
      ['Above', 'parent'],
      ['Beside', 'sibling'],
    ],
    first: 'Above',
    second: 'Beside',
  },
  {
    name: 'sibling to sibling',
    rows: [
      ['Above', 'parent'],
      ['Beside', 'sibling'],
      ['Other', 'sibling'],
    ],
    first: 'Beside',
    second: 'Other',
  },
  {
    name: 'child to jump, both away from the centre',
    rows: [
      ['Below', 'child'],
      ['Across', 'jump'],
    ],
    first: 'Below',
    second: 'Across',
  },
]

/** What the record of each end of a line says, and what the picture draws for it. */
const RECORD_CASES: readonly {
  readonly name: string
  readonly first: Stated | null
  readonly second: Stated | null
  readonly arrows: readonly ('from' | 'to')[]
  readonly labels: readonly { text: string; end?: 'from' | 'to' }[]
}[] = [
  {
    name: 'only the first note writes, with an arrow',
    first: ['onward', 'arrow'],
    second: null,
    arrows: ['to'],
    labels: [{ text: 'onward', end: 'to' }],
  },
  {
    name: 'only the second note writes, with an arrow',
    first: null,
    second: ['backward', 'arrow'],
    arrows: ['from'],
    labels: [{ text: 'backward', end: 'from' }],
  },
  {
    name: 'both notes write, each with an arrow',
    first: ['onward', 'arrow'],
    second: ['backward', 'arrow'],
    arrows: ['to', 'from'],
    labels: [
      { text: 'onward', end: 'to' },
      { text: 'backward', end: 'from' },
    ],
  },
  {
    name: 'the first note writes an arrow with no words',
    first: ['', 'arrow'],
    second: null,
    arrows: ['to'],
    labels: [],
  },
  {
    name: 'only the first note writes words, with no arrow',
    first: ['onward', null],
    second: null,
    arrows: [],
    labels: [{ text: 'onward' }],
  },
  {
    name: 'only the second note writes words, with no arrow',
    first: null,
    second: ['backward', null],
    arrows: [],
    labels: [{ text: 'backward' }],
  },
  {
    name: 'one note writes an arrow and the other words',
    first: ['onward', 'arrow'],
    second: ['backward', null],
    arrows: ['to'],
    labels: [{ text: 'onward', end: 'to' }, { text: 'backward' }],
  },
  {
    name: 'neither note writes anything',
    first: null,
    second: null,
    arrows: [],
    labels: [],
  },
]

describe.each(LINE_KINDS)('the line from $name', ({ rows, first, second }) => {
  it.each(RECORD_CASES)('draws it when $name', ({ first: one, second: other, arrows, labels }) => {
    const { plex, note } = renderImage(
      createNeighbourhood('Here', rows, [createLine(first, second, one, other)]),
    )

    expect(plex.edges).toHaveLength(1)
    const edge = plex.edges[0]!
    expect([note(edge.from), note(edge.to)]).toEqual([first, second])
    expect(edge.arrows ?? []).toEqual(arrows)
    expect(edge.labels ?? []).toEqual(labels)
  })

  it('draws one line for each connection of the neighbourhood and no other', () => {
    const lines = [createLine(first, second, ['onward', 'arrow']), createLine('Here', 'Isolated')]
    const { plex } = renderImage(createNeighbourhood('Here', rows, lines))

    expect(plex.edges).toHaveLength(1)
  })
})

describe('the words and arrowheads of one line', () => {
  it('draws two records with the same words in the same place once', () => {
    const [edge] = renderImage(
      createNeighbourhood(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', null], ['near', null])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([{ text: 'near' }])
  })

  it('draws the same words at both ends where each end has its own arrow', () => {
    const [edge] = renderImage(
      createNeighbourhood(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', 'arrow'], ['near', 'arrow'])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([
      { text: 'near', end: 'to' },
      { text: 'near', end: 'from' },
    ])
  })
})

describe('what each note of a neighbourhood is', () => {
  it('is what the vault says of the focus and of every note around it', () => {
    const neighbourhood: Neighbourhood = {
      focus: { path: 'Words.md', title: 'Words' },
      focusType: 'sample',
      related: [{ path: 'Forms.md', title: 'Forms', type: 'form', relation: 'child' }],
      connections: [],
    }

    expect([...getNoteTypes(neighbourhood)]).toStrictEqual([
      ['Words.md', 'sample'],
      ['Forms.md', 'form'],
    ])
  })

  it('is empty for a focus with no path', () => {
    const neighbourhood: Neighbourhood = {
      focus: { path: '', title: '' },
      focusType: 'sample',
      related: [],
      connections: [],
    }

    expect(getNoteTypes(neighbourhood).size).toBe(0)
  })
})

/** One note around the focus, as the vault answers one. */
type Neighbour = Neighbourhood['related'][number]

describe('two neighbourhoods drawing one picture', () => {
  const target = (): Neighbourhood => ({
    focus: { path: 'Here.md', title: 'Here' },
    focusType: 'note',
    related: [
      { path: 'Above.md', title: 'Above', type: 'note', relation: 'parent' },
      { path: 'Beside.md', title: 'Beside', type: 'note', relation: 'sibling' },
    ],
    connections: [
      createLine('Above.md', 'Here.md', null, ['part of', null]),
      createLine('Above.md', 'Beside.md', ['next to', null]),
    ],
  })

  /** The neighbourhood held, its second note changed as given. */
  const changed = (more: Partial<Neighbour>): Neighbourhood => {
    const one = target()
    return { ...one, related: [one.related[0]!, { ...one.related[1]!, ...more }] }
  }

  /** The neighbourhood held, its connection at `index` replaced. */
  const withLine = (index: number, line: Connection): Neighbourhood => {
    const one = target()
    return {
      ...one,
      connections: one.connections.map((current, position) =>
        position === index ? line : current,
      ),
    }
  }

  it('are two answers with every heading the same', () => {
    expect(areNeighbourhoodsEqual(target(), target())).toBe(true)
  })

  it('are two answers of nothing', () => {
    expect(areNeighbourhoodsEqual(null, null)).toBe(true)
  })

  it('are not an answer and nothing, either way round', () => {
    expect(areNeighbourhoodsEqual(target(), null)).toBe(false)
    expect(areNeighbourhoodsEqual(null, target())).toBe(false)
  })

  it('are not two answers standing on different notes', () => {
    const other = { ...target(), focus: { path: 'There.md', title: 'Here' } }

    expect(areNeighbourhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers calling the focus differently', () => {
    const other = { ...target(), focus: { path: 'Here.md', title: 'There' } }

    expect(areNeighbourhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers saying the focus is something different', () => {
    expect(areNeighbourhoodsEqual(target(), { ...target(), focusType: 'sample' })).toBe(false)
  })

  it('are not two answers with a different number of notes around', () => {
    const other = { ...target(), related: target().related.slice(0, 1) }

    expect(areNeighbourhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers with a different number of lines', () => {
    const other = { ...target(), connections: target().connections.slice(0, 1) }

    expect(areNeighbourhoodsEqual(target(), other)).toBe(false)
  })

  it.each<[string, Partial<Neighbour>]>([
    ['path', { path: 'Across.md' }],
    ['title', { title: 'Across' }],
    ['type', { type: 'sample' }],
    ['relation', { relation: 'child' }],
  ])('are not two answers whose second note differs in its %s', (_, more) => {
    expect(areNeighbourhoodsEqual(target(), changed(more))).toBe(false)
  })

  it.each<[string, Connection]>([
    ['first note', createLine('Other.md', 'Beside.md', ['next to', null])],
    ['second note', createLine('Above.md', 'Other.md', ['next to', null])],
    ['label of the first record', createLine('Above.md', 'Beside.md', ['far from', null])],
    ['marker of the first record', createLine('Above.md', 'Beside.md', ['next to', 'arrow'])],
    ['missing first record', createLine('Above.md', 'Beside.md')],
    ['second record', createLine('Above.md', 'Beside.md', ['next to', null], ['beside', null])],
    [
      'marker of the second record',
      createLine('Above.md', 'Beside.md', ['next to', null], ['', 'arrow']),
    ],
  ])('are not two answers whose line differs in its %s', (_, line) => {
    expect(areNeighbourhoodsEqual(target(), withLine(1, line))).toBe(false)
  })
})
