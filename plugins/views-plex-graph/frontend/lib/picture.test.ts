import { describe, expect, it } from 'vitest'
import type { Connection, ConnectionSide, LinkMarker } from '@numen/plugin-sdk'
import type { NoteGraph, NoteRelation } from '../types'
import { areNoteGraphsEqual, convertNoteGraph, getNames, getNoteTypes, getTabKinds } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/** A related note: where it is, and how it sits relative to the focus. */
type RelatedNoteRow = [path: string, relation: NoteRelation]

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

const createNoteGraph = (
  focus: string,
  rows: RelatedNoteRow[],
  lines: Connection[] = [],
): NoteGraph => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, relation]) => ({ path, title: path, type: 'note', relation })),
  connections: lines,
})

const renderImage = (noteGraph: NoteGraph) => {
  const map = createNodeIdMap()
  return { plex: convertNoteGraph(noteGraph, map.getNodeId), note: map.getNodePath }
}

const getEdges = (noteGraph: NoteGraph) => {
  const { plex, note } = renderImage(noteGraph)
  return plex.edges.map(
    (edge) =>
      `${note(edge.from)} -> ${note(edge.to)}${edge.labels ? ` (${edge.labels.map((label) => label.text).join(', ')})` : ''}`,
  )
}

describe('what the plex is handed', () => {
  it('runs an edge the way the connection runs', () => {
    expect(
      getEdges(
        createNoteGraph(
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
      createNoteGraph('Here', [['Below', 'child']], [createLine('Here', 'Below', ['', null])]),
    ).plex.edges
    expect(line?.labels).toBeUndefined()
  })

  it('draws a line from the connection that joins its notes, whichever notes they are', () => {
    expect(
      getEdges(
        createNoteGraph(
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
    const plex = convertNoteGraph(
      createNoteGraph('Here', [['Beside', 'sibling']], [createLine('Missing', 'Beside')]),
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
      createNoteGraph('Here', [
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
  const target = createNoteGraph(
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
    const plex = convertNoteGraph(target, (path) => `#${path}`)

    expect(plex.nodes.map((node) => node.id)).toEqual(['#Here', '#Above', '#Below', '#Beside'])
    expect(map.getNodePath('Here')).toBe('')
  })

  it('calls every note of one picture something different from every other', () => {
    const { plex } = renderImage(target)

    expect(new Set(plex.nodes.map((node) => node.id)).size).toBe(plex.nodes.length)
  })

  it('runs an edge between the tickets of the two notes it joins', () => {
    const plex = convertNoteGraph(target, (path) => `#${path}`)

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
  readonly rows: RelatedNoteRow[]
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
      createNoteGraph('Here', rows, [createLine(first, second, one, other)]),
    )

    expect(plex.edges).toHaveLength(1)
    const edge = plex.edges[0]!
    expect([note(edge.from), note(edge.to)]).toEqual([first, second])
    expect(edge.arrows ?? []).toEqual(arrows)
    expect(edge.labels ?? []).toEqual(labels)
  })

  it('draws one line for each connection of the noteGraph and no other', () => {
    const lines = [createLine(first, second, ['onward', 'arrow']), createLine('Here', 'Isolated')]
    const { plex } = renderImage(createNoteGraph('Here', rows, lines))

    expect(plex.edges).toHaveLength(1)
  })
})

describe('the words and arrowheads of one line', () => {
  it('draws two records with the same words once', () => {
    const [edge] = renderImage(
      createNoteGraph(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', null], ['near', null])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([{ text: 'near' }])
  })

  it('draws the same words once where each end has its own arrow', () => {
    const [edge] = renderImage(
      createNoteGraph(
        'Here',
        [['Across', 'jump']],
        [createLine('Across', 'Here', ['near', 'arrow'], ['near', 'arrow'])],
      ),
    ).plex.edges

    expect(edge?.labels).toEqual([{ text: 'near', end: 'to' }])
  })
})

describe('what each note of a noteGraph is', () => {
  it('is what the vault says of the focus and of every note around it', () => {
    const noteGraph: NoteGraph = {
      focus: { path: 'Words.md', title: 'Words' },
      focusType: 'sample',
      related: [{ path: 'Forms.md', title: 'Forms', type: 'form', relation: 'child' }],
      connections: [],
    }

    expect([...getNoteTypes(noteGraph)]).toStrictEqual([
      ['Words.md', 'sample'],
      ['Forms.md', 'form'],
    ])
  })

  it('is empty for a focus with no path', () => {
    const noteGraph: NoteGraph = {
      focus: { path: '', title: '' },
      focusType: 'sample',
      related: [],
      connections: [],
    }

    expect(getNoteTypes(noteGraph).size).toBe(0)
  })
})

/** One note around the focus, as the vault answers one. */
type RelatedNote = NoteGraph['related'][number]

describe('two note graphs drawing one picture', () => {
  const target = (): NoteGraph => ({
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

  /** The noteGraph held, its second note changed as given. */
  const changed = (more: Partial<RelatedNote>): NoteGraph => {
    const one = target()
    return { ...one, related: [one.related[0]!, { ...one.related[1]!, ...more }] }
  }

  /** The noteGraph held, its connection at `index` replaced. */
  const withLine = (index: number, line: Connection): NoteGraph => {
    const one = target()
    return {
      ...one,
      connections: one.connections.map((current, position) =>
        position === index ? line : current,
      ),
    }
  }

  it('are two answers with every heading the same', () => {
    expect(areNoteGraphsEqual(target(), target())).toBe(true)
  })

  it('are two answers of nothing', () => {
    expect(areNoteGraphsEqual(null, null)).toBe(true)
  })

  it('are not an answer and nothing, either way round', () => {
    expect(areNoteGraphsEqual(target(), null)).toBe(false)
    expect(areNoteGraphsEqual(null, target())).toBe(false)
  })

  it('are not two answers standing on different notes', () => {
    const other = { ...target(), focus: { path: 'There.md', title: 'Here' } }

    expect(areNoteGraphsEqual(target(), other)).toBe(false)
  })

  it('are not two answers calling the focus differently', () => {
    const other = { ...target(), focus: { path: 'Here.md', title: 'There' } }

    expect(areNoteGraphsEqual(target(), other)).toBe(false)
  })

  it('are not two answers saying the focus is something different', () => {
    expect(areNoteGraphsEqual(target(), { ...target(), focusType: 'sample' })).toBe(false)
  })

  it('are not two answers with a different number of notes around', () => {
    const other = { ...target(), related: target().related.slice(0, 1) }

    expect(areNoteGraphsEqual(target(), other)).toBe(false)
  })

  it('are not two answers with a different number of lines', () => {
    const other = { ...target(), connections: target().connections.slice(0, 1) }

    expect(areNoteGraphsEqual(target(), other)).toBe(false)
  })

  it.each<[string, Partial<RelatedNote>]>([
    ['path', { path: 'Across.md' }],
    ['title', { title: 'Across' }],
    ['type', { type: 'sample' }],
    ['relation', { relation: 'child' }],
  ])('are not two answers whose second note differs in its %s', (_, more) => {
    expect(areNoteGraphsEqual(target(), changed(more))).toBe(false)
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
    expect(areNoteGraphsEqual(target(), withLine(1, line))).toBe(false)
  })
})

/** An item of depth 2: a note a related note is related to, with its relation to that note. */
type ItemRow = [path: string, relation: NoteRelation, relatedTo: string]

const createDeepNoteGraph = (
  focus: string,
  rows: RelatedNoteRow[],
  items: ItemRow[],
  lines: Connection[] = [],
): NoteGraph => {
  const base = createNoteGraph(focus, rows, lines)
  return {
    ...base,
    related: [
      ...base.related,
      ...items.map(([path, relation, relatedTo]) => ({
        path,
        title: path,
        type: 'note',
        relation,
        relatedTo,
      })),
    ],
  }
}

const token = (path: string) => `id:${path}`

const describeNames = (noteGraph: NoteGraph, limit: number, path: string) => {
  const names = getNames(noteGraph, limit, token).get(token(path))
  return {
    above: names?.above.map((one) => one.title) ?? [],
    below: names?.below.map((one) => one.title) ?? [],
  }
}

const crowdedGraph = (): NoteGraph =>
  createDeepNoteGraph(
    'Here',
    [['Kid', 'child']],
    [
      ['Jump1', 'jump', 'Kid'],
      ['Kid1', 'child', 'Kid'],
      ['Up1', 'parent', 'Kid'],
      ['Kid2', 'child', 'Kid'],
      ['Up2', 'parent', 'Kid'],
      ['Jump2', 'jump', 'Kid'],
      ['Kid3', 'child', 'Kid'],
      ['Up3', 'parent', 'Kid'],
      ['Kid4', 'child', 'Kid'],
    ],
  )

describe('the names under and over a related note', () => {
  it('puts the parents over the note and the rest under it, parents first, then children, then jumps', () => {
    expect(describeNames(crowdedGraph(), 12, 'Kid')).toEqual({
      above: ['Up1', 'Up2', 'Up3'],
      below: ['Kid1', 'Kid2', 'Kid3', 'Kid4', 'Jump1', 'Jump2'],
    })
  })

  it('cuts the names to the limit in that order, parents counting first', () => {
    expect(describeNames(crowdedGraph(), 6, 'Kid')).toEqual({
      above: ['Up1', 'Up2', 'Up3'],
      below: ['Kid1', 'Kid2', 'Kid3'],
    })
    expect(describeNames(crowdedGraph(), 1, 'Kid')).toEqual({ above: ['Up1'], below: [] })
  })

  it('keeps no entry for a note with no names, and none for the focus', () => {
    const names = getNames(
      createDeepNoteGraph('Here', [['Kid', 'child'], ['Lone', 'child']], [['Leaf', 'child', 'Kid']]),
      6,
      token,
    )

    expect([...names.keys()]).toEqual([token('Kid')])
  })

  it('names a note under every note it is related to', () => {
    const names = getNames(
      createDeepNoteGraph(
        'Here',
        [['A', 'child'], ['B', 'child']],
        [['Shared', 'child', 'A'], ['Shared', 'child', 'B']],
      ),
      6,
      token,
    )

    expect(names.get(token('A'))?.below.map((one) => one.title)).toEqual(['Shared'])
    expect(names.get(token('B'))?.below.map((one) => one.title)).toEqual(['Shared'])
  })

  it('gives a name the id its note has, so choosing it reaches the note', () => {
    const names = getNames(crowdedGraph(), 12, token)

    expect(names.get(token('Kid'))?.above.map((one) => one.id)).toEqual([
      token('Up1'),
      token('Up2'),
      token('Up3'),
    ])
  })
})

describe('the picture of a note graph with the notes related to related notes', () => {
  it('draws the nodes and edges of depth 1 and no node or edge for a name', () => {
    const lines = [createLine('Here', 'Kid')]
    const one = createNoteGraph('Here', [['Kid', 'child']], lines)
    const two = createDeepNoteGraph('Here', [['Kid', 'child']], [['Leaf', 'child', 'Kid']], lines)

    const map = createNodeIdMap()
    const plexTwo = convertNoteGraph(two, map.getNodeId)
    const plexOne = convertNoteGraph(one, createNodeIdMap().getNodeId)

    expect(plexTwo).toEqual(plexOne)
    expect(plexTwo.nodes.map((node) => node.title)).not.toContain('Leaf')
  })

  it('draws a note that is a name of one note and related to the focus once', () => {
    const noteGraph = createDeepNoteGraph(
      'Here',
      [['A', 'child'], ['B', 'child']],
      [['B', 'child', 'A']],
    )

    const titles = convertNoteGraph(noteGraph, createNodeIdMap().getNodeId).nodes.map((n) => n.title)

    expect(titles).toEqual(['Here', 'A', 'B'])
  })

  it('still reports the type and tab kind of an item', () => {
    const noteGraph = createDeepNoteGraph('Here', [['Kid', 'child']], [['Leaf', 'child', 'Kid']])
    const typed: NoteGraph = {
      ...noteGraph,
      related: noteGraph.related.map((one) =>
        one.path === 'Leaf' ? { ...one, type: 'sample', tabKind: 'sample' } : one,
      ),
    }

    expect(getNoteTypes(typed).get('Leaf')).toBe('sample')
    expect(getTabKinds(typed).get('Leaf')).toBe('sample')
  })
})

describe('two note graphs that differ only in the note an item belongs to', () => {
  it('are not equal', () => {
    const one = createDeepNoteGraph('Here', [['A', 'child'], ['B', 'child']], [['Leaf', 'child', 'A']])
    const other = createDeepNoteGraph('Here', [['A', 'child'], ['B', 'child']], [['Leaf', 'child', 'B']])

    expect(areNoteGraphsEqual(one, other)).toBe(false)
    expect(areNoteGraphsEqual(one, createDeepNoteGraph('Here', [['A', 'child'], ['B', 'child']], [['Leaf', 'child', 'A']]))).toBe(true)
  })
})

describe('references in the picture', () => {
  type Direction = 'outgoing' | 'incoming' | 'both'
  type Mode = 'off' | 'outgoing' | 'incoming' | 'both'

  const withReferences = (): NoteGraph => ({
    focus: { path: 'Here.md', title: 'Here' },
    focusType: 'note',
    related: [
      { path: 'Below.md', title: 'Below', type: 'note', relation: 'child' },
      {
        path: 'Cites.md',
        title: 'Cites',
        type: 'note',
        relation: 'reference',
        direction: 'outgoing',
      },
      {
        path: 'CitedBy.md',
        title: 'CitedBy',
        type: 'note',
        relation: 'reference',
        direction: 'incoming',
      },
      {
        path: 'Mutual.md',
        title: 'Mutual',
        type: 'note',
        relation: 'reference',
        direction: 'both',
      },
    ],
    connections: [createLine('Here.md', 'Below.md', ['part', null])],
  })

  const draw = (noteGraph: NoteGraph, mode: Mode) => {
    const map = createNodeIdMap()
    const plex = (
      convertNoteGraph as (
        g: NoteGraph,
        token: (path: string) => string,
        mode: Mode,
      ) => ReturnType<typeof convertNoteGraph>
    )(noteGraph, map.getNodeId, mode)
    return { plex, note: map.getNodePath }
  }

  const getShownNodes = (mode: Mode): string[] =>
    draw(withReferences(), mode)
      .plex.nodes.filter((node) => node.relation === 'reference')
      .map((node) => node.title)

  it.each<[Mode, string[]]>([
    ['off', []],
    ['outgoing', ['Cites', 'Mutual']],
    ['incoming', ['CitedBy', 'Mutual']],
    ['both', ['Cites', 'CitedBy', 'Mutual']],
  ])('shows with the setting %s the references %j', (mode, titles) => {
    expect(getShownNodes(mode)).toEqual(titles)
  })

  it.each<Mode>(['off', 'outgoing', 'incoming', 'both'])(
    'keeps the child with the setting %s',
    (mode) => {
      const { plex } = draw(withReferences(), mode)

      expect(plex.nodes.map((node) => node.title)).toEqual(
        expect.arrayContaining(['Here', 'Below']),
      )
    },
  )

  it('points the line at the note the focus links to, and at the focus from the note that links to it', () => {
    const { plex, note } = draw(withReferences(), 'both')
    const referenceEdges = plex.edges.filter(
      (edge) => note(edge.from) !== 'Below.md' && note(edge.to) !== 'Below.md',
    )

    expect(referenceEdges.map((edge) => [note(edge.from), note(edge.to), edge.arrows])).toEqual([
      ['Here.md', 'Cites.md', ['to']],
      ['Here.md', 'CitedBy.md', ['from']],
      ['Here.md', 'Mutual.md', ['from', 'to']],
    ])
    for (const edge of referenceEdges) expect(edge.labels).toBeUndefined()
  })

  it('draws the lines of the connections before the lines of the references', () => {
    const { plex, note } = draw(withReferences(), 'both')

    expect(note(plex.edges[0]!.to)).toBe('Below.md')
    expect(plex.edges).toHaveLength(4)
  })

  it('draws no line for a reference the setting hides', () => {
    const { plex } = draw(withReferences(), 'off')

    expect(plex.edges).toHaveLength(1)
  })

  it('draws a reference with no direction without an arrow, in the setting both only', () => {
    const noteGraph = withReferences()
    const bare: NoteGraph = {
      ...noteGraph,
      related: [{ path: 'Bare.md', title: 'Bare', type: 'note', relation: 'reference' }],
      connections: [],
    }

    const both = draw(bare, 'both')
    expect(both.plex.nodes.map((node) => node.title)).toContain('Bare')
    expect(both.plex.edges).toHaveLength(1)
    expect(both.plex.edges[0]!.arrows).toBeUndefined()
    for (const mode of ['off', 'outgoing', 'incoming'] as const) {
      expect(draw(bare, mode).plex.nodes.map((node) => node.title)).not.toContain('Bare')
    }
  })

  it('joins no reference to any note but the focus', () => {
    const { plex, note } = draw(withReferences(), 'both')

    for (const edge of plex.edges.filter(
      (one) => note(one.from) !== 'Below.md' && note(one.to) !== 'Below.md',
    )) {
      expect([note(edge.from), note(edge.to)]).toContain('Here.md')
    }
  })

  const withDirection = (related: RelatedNote, value: Direction | undefined): RelatedNote => {
    const { direction: _previous, ...rest } = related
    return value === undefined ? rest : { ...rest, direction: value }
  }

  const direction = (value: Direction | undefined): NoteGraph => {
    const one = withReferences()
    return {
      ...one,
      related: one.related.map((related) =>
        related.path === 'Cites.md' ? withDirection(related, value) : related,
      ),
    }
  }

  it('is not equal to a note graph whose reference is linked the other way', () => {
    expect(areNoteGraphsEqual(direction('outgoing'), direction('incoming'))).toBe(false)
    expect(areNoteGraphsEqual(direction('outgoing'), direction('both'))).toBe(false)
    expect(areNoteGraphsEqual(direction('outgoing'), direction(undefined))).toBe(false)
  })

  it('is equal to a note graph whose reference is linked the same way', () => {
    expect(areNoteGraphsEqual(direction('incoming'), direction('incoming'))).toBe(true)
  })
})
