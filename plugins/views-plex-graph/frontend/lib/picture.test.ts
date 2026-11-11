import { describe, expect, it } from 'vitest'
import type { Connection, ConnectionSide, LinkMarker } from '@numen/plugin-sdk'
import type { Neighborhood, NoteRelation } from '../types'
import { areNeighborhoodsEqual, convertNeighborhood, getNoteTypes } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/** A related note: where it is, and how it sits relative to the focus. */
type NeighborRow = [path: string, relation: NoteRelation]

/** A record one note writes about the other: its label and what it draws at the other note. */
type Stated = readonly [label: string, marker: LinkMarker | null]

const toSide = (record: Stated | null): ConnectionSide | null =>
  record === null ? null : { label: record[0], marker: record[1], reason: '' }

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

const createNeighborhood = (
  focus: string,
  rows: NeighborRow[],
  lines: Connection[] = [],
): Neighborhood => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, relation]) => ({ path, title: path, type: 'note', relation })),
  connections: lines,
})

const renderImage = (neighborhood: Neighborhood) => {
  const map = createNodeIdMap()
  return { plex: convertNeighborhood(neighborhood, map.getNodeId), note: map.getNodePath }
}

const getEdges = (neighborhood: Neighborhood) => {
  const { plex, note } = renderImage(neighborhood)
  return plex.edges.map(
    (edge) =>
      `${note(edge.from)} -> ${note(edge.to)}${edge.labels ? ` (${edge.labels.map((label) => label.text).join(', ')})` : ''}`,
  )
}

describe('what the plex is handed', () => {
  it('runs an edge the way the connection runs', () => {
    expect(
      getEdges(
        createNeighborhood(
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
      createNeighborhood('Here', [['Below', 'child']], [createLine('Here', 'Below', ['', null])]),
    ).plex.edges
    expect(line?.labels).toBeUndefined()
  })

  it('draws a line from the connection that joins its notes, whichever notes they are', () => {
    expect(
      getEdges(
        createNeighborhood(
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
    const plex = convertNeighborhood(
      createNeighborhood('Here', [['Beside', 'sibling']], [createLine('Missing', 'Beside')]),
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
      createNeighborhood('Here', [
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
  const target = createNeighborhood(
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
    const plex = convertNeighborhood(target, (path) => `#${path}`)

    expect(plex.nodes.map((node) => node.id)).toEqual(['#Here', '#Above', '#Below', '#Beside'])
    expect(map.getNodePath('Here')).toBe('')
  })

  it('calls every note of one picture something different from every other', () => {
    const { plex } = renderImage(target)

    expect(new Set(plex.nodes.map((node) => node.id)).size).toBe(plex.nodes.length)
  })

  it('runs an edge between the tickets of the two notes it joins', () => {
    const plex = convertNeighborhood(target, (path) => `#${path}`)

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
  readonly rows: NeighborRow[]
  readonly first: string
  readonly second: string
}[] = [
  { name: 'center to parent', rows: [['Above', 'parent']], first: 'Above', second: 'Here' },
  { name: 'center to child', rows: [['Below', 'child']], first: 'Here', second: 'Below' },
  { name: 'center to jump', rows: [['Across', 'jump']], first: 'Across', second: 'Here' },
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
    name: 'child to jump, both away from the center',
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
      createNeighborhood('Here', rows, [createLine(first, second, one, other)]),
    )

    expect(plex.edges).toHaveLength(1)
    const edge = plex.edges[0]!
    expect([note(edge.from), note(edge.to)]).toEqual([first, second])
    expect(edge.arrows ?? []).toEqual(arrows)
    expect(edge.labels ?? []).toEqual(labels)
  })

  it('draws one line for each connection of the neighborhood and no other', () => {
    const lines = [createLine(first, second, ['onward', 'arrow']), createLine('Here', 'Isolated')]
    const { plex } = renderImage(createNeighborhood('Here', rows, lines))

    expect(plex.edges).toHaveLength(1)
  })
})

describe('the words and arrowheads of one line', () => {
  it('draws two records with the same words once', () => {
    const [edge] = renderImage(
      createNeighborhood(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', null], ['near', null])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([{ text: 'near' }])
  })

  it('draws the same words once where each end has its own arrow', () => {
    const [edge] = renderImage(
      createNeighborhood(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', 'arrow'], ['near', 'arrow'])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([{ text: 'near', end: 'to' }])
  })
})

describe('what each note of a neighborhood is', () => {
  it('is what the vault says of the focus and of every note around it', () => {
    const neighborhood: Neighborhood = {
      focus: { path: 'Words.md', title: 'Words' },
      focusType: 'sample',
      related: [{ path: 'Forms.md', title: 'Forms', type: 'form', relation: 'child' }],
      connections: [],
    }

    expect([...getNoteTypes(neighborhood)]).toStrictEqual([
      ['Words.md', 'sample'],
      ['Forms.md', 'form'],
    ])
  })

  it('is empty for a focus with no path', () => {
    const neighborhood: Neighborhood = {
      focus: { path: '', title: '' },
      focusType: 'sample',
      related: [],
      connections: [],
    }

    expect(getNoteTypes(neighborhood).size).toBe(0)
  })
})

/** One note around the focus, as the vault answers one. */
type Neighbor = Neighborhood['related'][number]

describe('two neighborhoods drawing one picture', () => {
  const target = (): Neighborhood => ({
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

  /** The neighborhood held, its second note changed as given. */
  const changed = (more: Partial<Neighbor>): Neighborhood => {
    const one = target()
    return { ...one, related: [one.related[0]!, { ...one.related[1]!, ...more }] }
  }

  /** The neighborhood held, its connection at `index` replaced. */
  const withLine = (index: number, line: Connection): Neighborhood => {
    const one = target()
    return {
      ...one,
      connections: one.connections.map((current, position) =>
        position === index ? line : current,
      ),
    }
  }

  it('are two answers with every heading the same', () => {
    expect(areNeighborhoodsEqual(target(), target())).toBe(true)
  })

  it('are two answers of nothing', () => {
    expect(areNeighborhoodsEqual(null, null)).toBe(true)
  })

  it('are not an answer and nothing, either way round', () => {
    expect(areNeighborhoodsEqual(target(), null)).toBe(false)
    expect(areNeighborhoodsEqual(null, target())).toBe(false)
  })

  it('are not two answers standing on different notes', () => {
    const other = { ...target(), focus: { path: 'There.md', title: 'Here' } }

    expect(areNeighborhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers calling the focus differently', () => {
    const other = { ...target(), focus: { path: 'Here.md', title: 'There' } }

    expect(areNeighborhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers saying the focus is something different', () => {
    expect(areNeighborhoodsEqual(target(), { ...target(), focusType: 'sample' })).toBe(false)
  })

  it('are not two answers with a different number of notes around', () => {
    const other = { ...target(), related: target().related.slice(0, 1) }

    expect(areNeighborhoodsEqual(target(), other)).toBe(false)
  })

  it('are not two answers with a different number of lines', () => {
    const other = { ...target(), connections: target().connections.slice(0, 1) }

    expect(areNeighborhoodsEqual(target(), other)).toBe(false)
  })

  it.each<[string, Partial<Neighbor>]>([
    ['path', { path: 'Across.md' }],
    ['title', { title: 'Across' }],
    ['type', { type: 'sample' }],
    ['relation', { relation: 'child' }],
  ])('are not two answers whose second note differs in its %s', (_, more) => {
    expect(areNeighborhoodsEqual(target(), changed(more))).toBe(false)
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
    expect(areNeighborhoodsEqual(target(), withLine(1, line))).toBe(false)
  })
})
