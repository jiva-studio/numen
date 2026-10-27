import { describe, expect, it } from 'vitest'
import type { Neighbourhood, NoteRelation } from '../types'
import { areNeighbourhoodsEqual, convertNeighbourhood, getNoteTypes } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/** A note and what sits around it: relation, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type NeighbourRow = [string, NoteRelation, string, string, boolean?]

const createNeighbourhood = (focus: string, rows: NeighbourRow[]): Neighbourhood => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, relation, label, through, mutual]) => ({
    path,
    title: path,
    type: 'note',
    relation,
    label,
    through,
    isMutual: mutual ?? false,
  })),
})

/**
 * The picture, and the note each node ID in it stands for. Every note here is
 * one carrying no identifier, which is the note the picture has to draw.
 */
const renderPicture = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const map = createNodeIdMap()
  return { plex: convertNeighbourhood(neighbourhood, map.getNodeId), note: map.getNodePath }
}

/**
 * Every edge, as the notes at its two ends. A token no note holds reads as
 * nothing, so an edge drawn under anything else says so here.
 */
const getEdges = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = renderPicture(neighbourhood)
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
    // A line carries only the word the person put on it.
    const [line] = renderPicture(createNeighbourhood('Here', [['Below', 'child', '', '']])).plex
      .edges
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
    // Each sibling belongs to one of the two parents, and only the answer
    // knows which. A line drawn from the other one says a note is the child of
    // something the vault never joined it to.
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
    // It would otherwise fall back to the focus, which says the wrong thing:
    // a sibling is not a child.
    expect(getEdges(createNeighbourhood('Here', [['Beside', 'sibling', '', 'Missing']]))).toEqual(
      [],
    )
  })

  it('places every note it was given', () => {
    const { plex, note } = renderPicture(
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
    const { plex } = renderPicture(target)

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

/** Every edge as the end its arrow is drawn at, and the pair it joins. */
const arrows = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = renderPicture(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}: ${edge.arrow ?? 'no arrow'}`,
  )
}

describe('whose wording is on the line', () => {
  it('points the arrow out of the focus, down a line into a child', () => {
    // Marrowfield allotments relations Alice Fenn below it and writes a word on the
    // link; Alice Fenn names the same relationship back.
    expect(
      arrows(
        createNeighbourhood('Marrowfield allotments', [
          ['Alice Fenn, The Chair', 'child', 'the chair since May', '', true],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: to'])
  })

  it('points it out of the focus down a line the relationship runs into', () => {
    // The same link from the other end, where Alice Fenn wrote nothing on it.
    // An arrow says who chose the wording, whether or not there is any.
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Marrowfield allotments', 'parent', '', '', true],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: from'])
  })

  it('points it out of the focus on a jump each note wrote its own word on', () => {
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Bram Doyle', 'jump', 'the oldest tenant', '', true],
        ]),
      ),
    ).toEqual(['Bram Doyle -> Alice Fenn, The Chair: from'])
  })

  it('draws no arrow where only one note named a jump relationship', () => {
    // Alice Fenn jumps to Cora Hale and Cora Hale says nothing back:
    // an associative jump with one note naming it has no arrow.
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Cora Hale', 'jump', 'keys, hoses, the gate', '', false],
        ]),
      ),
    ).toEqual(['Cora Hale -> Alice Fenn, The Chair: no arrow'])
  })

  it('draws no arrow for single child and parent relationships by default', () => {
    // Single child link has no arrow by default
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Child note', 'child', 'subtask', '', false],
        ]),
      ),
    ).toEqual(['Alice Fenn, The Chair -> Child note: no arrow'])

    // Single parent link has no arrow by default
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Parent note', 'parent', 'depends on', '', false],
        ]),
      ),
    ).toEqual(['Parent note -> Alice Fenn, The Chair: no arrow'])
  })

  it('draws no arrow on a sibling, whose line is between two other notes', () => {
    // The line hangs off the shared parent, so neither end of it is the note
    // in focus and there is no end for an arrow to name.
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Marrowfield allotments', 'parent', '', '', true],
          ['The question of the rota', 'sibling', '', 'Marrowfield allotments', true],
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
      },
      {
        path: 'Beside.md',
        title: 'Beside',
        type: 'note',
        relation: 'sibling',
        label: 'next to',
        through: 'Above.md',
        isMutual: true,
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
    ['isMutual', { isMutual: false }],
  ])('are not two answers whose second note differs in its %s', (_, more) => {
    expect(areNeighbourhoodsEqual(target(), changed(more))).toBe(false)
  })
})
