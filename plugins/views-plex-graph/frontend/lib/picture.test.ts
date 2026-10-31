import { describe, expect, it } from 'vitest'
import type { LinkMarker } from '@numen/plugin-sdk'
import type { Neighbourhood, NoteRelation } from '../types'
import { areNeighbourhoodsEqual, convertNeighbourhood, getNoteTypes } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/**
 * A related note: relation, label, the note it connects through, and the
 * markers of the record the focus writes and of the record the note writes.
 */
type NeighbourRow = [string, NoteRelation, string, string, [LinkMarker | null, LinkMarker | null]?]

const createNeighbourhood = (focus: string, rows: NeighbourRow[]): Neighbourhood => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, relation, label, through, markers]) => ({
    path,
    title: path,
    type: 'note',
    relation,
    label,
    through,
    isMutual: false,
    marker: markers?.[0] ?? null,
    reverseMarker: markers?.[1] ?? null,
  })),
})

/**
 * The graph, and the note each node ID in it identifies. Every note here has no
 * identifier, which is the case the graph has to render.
 */
const renderImage = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const map = createNodeIdMap()
  return { plex: convertNeighbourhood(neighbourhood, map.getNodeId), note: map.getNodePath }
}

/**
 * Every edge, as the notes at its two ends. A token that no note holds resolves
 * to nothing, so an edge built from any other token is visible here.
 */
const getEdges = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = renderImage(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}${edge.label ? ` (${edge.label})` : ''}`,
  )
}

describe('what the plex is handed', () => {
  it('runs an edge the way the relationship runs', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', 'parent', 'part of', ''],
          ['Below', 'child', '', ''],
          ['Across', 'jump', 'see also', ''],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Here -> Below', 'Across -> Here (see also)'])
  })

  it('writes nothing on a line the person wrote nothing on', () => {
    // A line carries only the label the user put on it.
    const [line] = renderImage(createNeighbourhood('Here', [['Below', 'child', '', '']])).plex.edges
    expect(line?.label).toBeUndefined()
  })

  it('hangs a sibling off the parent, not off the focus', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', 'parent', 'part of', ''],
          ['Beside', 'sibling', '', 'Above'],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Above -> Beside'])
  })

  it('hangs each sibling off its own parent when there are two', () => {
    // Each sibling belongs to one of the two parents, and only the response
    // determines which. A line drawn from the other parent would assert a
    // parent-child relation the vault never recorded.
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Machine learning', 'parent', '', ''],
          ['Eigenvector', 'parent', 'needs', ''],
          ['Clustering', 'sibling', '', 'Machine learning'],
          ['SVD', 'sibling', '', 'Eigenvector'],
        ]),
      ),
    ).toEqual([
      'Machine learning -> Here',
      'Eigenvector -> Here (needs)',
      'Machine learning -> Clustering',
      'Eigenvector -> SVD',
    ])
  })

  it('draws no sibling edge when no parent is shown', () => {
    // A fallback to the focus is wrong here, because a sibling is not a child.
    expect(getEdges(createNeighbourhood('Here', [['Beside', 'sibling', '', 'Missing']]))).toEqual(
      [],
    )
  })

  it('places every note it was given', () => {
    const { plex, note } = renderImage(
      createNeighbourhood('Here', [
        ['Above', 'parent', 'part of', ''],
        ['Below', 'child', '', ''],
        ['Beside', 'sibling', '', 'Above'],
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
  const target = createNeighbourhood('Here', [
    ['Above', 'parent', '', ''],
    ['Below', 'child', '', ''],
    ['Beside', 'sibling', '', 'Above'],
  ])

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

  it('mints nothing for a parent the picture does not show', () => {
    // A sibling whose parent is off the screen hangs off nothing, and the note
    // it named is one this picture never drew.
    const request: string[] = []
    convertNeighbourhood(
      createNeighbourhood('Here', [['Beside', 'sibling', '', 'Missing']]),
      (path) => {
        request.push(path)
        return `#${path}`
      },
    )

    expect(request).not.toContain('Missing')
  })
})

/** Every edge as the ends its arrowheads are drawn at, and the pair it joins. */
const arrows = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = renderImage(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}: ${edge.arrows?.join(' and ') ?? 'no arrow'}`,
  )
}

describe('the markers on the line', () => {
  it('draws the arrow of the focus at the child, down a line out of the focus', () => {
    expect(
      arrows(
        createNeighbourhood('Marrowfield allotments', [
          ['Alice Fenn, The Chair', 'child', 'the chair since May', '', ['arrow', null]],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: to'])
  })

  it('draws the arrow of the child at the focus', () => {
    expect(
      arrows(
        createNeighbourhood('Marrowfield allotments', [
          ['Alice Fenn, The Chair', 'child', '', '', [null, 'arrow']],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: from'])
  })

  it('draws the arrow of the focus at a parent, down a line into the focus', () => {
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Marrowfield allotments', 'parent', '', '', ['arrow', null]],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: from'])
  })

  it('draws an arrow at each end where both records carry one', () => {
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Bram Doyle', 'jump', 'the oldest tenant', '', ['arrow', 'arrow']],
        ]),
      ),
    ).toEqual(['Bram Doyle -> Alice Fenn, The Chair: from and to'])
  })

  it('draws no arrow where neither record carries a marker, whoever wrote a record', () => {
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Cora Hale', 'jump', 'keys, hoses, the gate', '', [null, null]],
          ['Child note', 'child', 'subtask', ''],
          ['Parent note', 'parent', 'depends on', ''],
        ]),
      ),
    ).toEqual([
      'Cora Hale -> Alice Fenn, The Chair: no arrow',
      'Alice Fenn, The Chair -> Child note: no arrow',
      'Parent note -> Alice Fenn, The Chair: no arrow',
    ])
  })

  it('draws no arrow on a sibling, whose line is between two other notes', () => {
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Marrowfield allotments', 'parent', '', '', ['arrow', null]],
          ['The question of the rota', 'sibling', '', 'Marrowfield allotments', ['arrow', 'arrow']],
        ]),
      ),
    ).toEqual([
      'Marrowfield allotments -> Alice Fenn, The Chair: from',
      'Marrowfield allotments -> The question of the rota: no arrow',
    ])
  })
})

describe('a sibling nested under the note in focus', () => {
  it('runs its line from the focus, which the picture draws', () => {
    expect(getEdges(createNeighbourhood('Here', [['Beside', 'sibling', '', 'Here']]))).toEqual([
      'Here -> Beside',
    ])
  })
})

describe('what each note of a neighbourhood is', () => {
  it('is what the vault says of the focus and of every note around it', () => {
    const neighbourhood: Neighbourhood = {
      focus: { path: 'Words.md', title: 'Words' },
      focusType: 'sample',
      related: [
        {
          path: 'Forms.md',
          title: 'Forms',
          type: 'form',
          relation: 'child',
          label: '',
          through: '',
          isMutual: false,
          marker: null,
          reverseMarker: null,
        },
      ],
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
      {
        path: 'Above.md',
        title: 'Above',
        type: 'note',
        relation: 'parent',
        label: 'part of',
        through: '',
        isMutual: false,
        marker: null,
        reverseMarker: null,
      },
      {
        path: 'Beside.md',
        title: 'Beside',
        type: 'note',
        relation: 'sibling',
        label: 'next to',
        through: 'Above.md',
        isMutual: true,
        marker: null,
        reverseMarker: null,
      },
    ],
  })

  /** The neighbourhood held, its second note changed as given. */
  const changed = (more: Partial<Neighbour>): Neighbourhood => {
    const one = target()
    return { ...one, related: [one.related[0]!, { ...one.related[1]!, ...more }] }
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

  it.each<[string, Partial<Neighbour>]>([
    ['path', { path: 'Across.md' }],
    ['title', { title: 'Across' }],
    ['type', { type: 'sample' }],
    ['relation', { relation: 'child' }],
    ['label', { label: 'far from' }],
    ['through', { through: 'Here.md' }],
    ['marker', { marker: 'arrow' }],
    ['reverseMarker', { reverseMarker: 'arrow' }],
  ])('are not two answers whose second note differs in its %s', (_, more) => {
    expect(areNeighbourhoodsEqual(target(), changed(more))).toBe(false)
  })
})
