import { describe, expect, it } from 'vitest'
import type { Neighbourhood, Seat } from '@/entities/note'
import { asPlex } from './picture'
import { createNodeIdMap } from './nodeIdMap'

/** A note and what sits around it: seat, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type NeighbourRow = [string, Seat, string, string, boolean?]

const createNeighbourhood = (focus: string, rows: NeighbourRow[]): Neighbourhood => ({
  focus: { path: focus, title: focus },
  focusType: 'note',
  related: rows.map(([path, seat, label, through, mutual]) => ({
    path,
    title: path,
    type: 'note',
    seat,
    label,
    through,
    isMutual: mutual ?? false,
  })),
})

/**
 * The picture, and the note each node ID in it stands for. Every note here is
 * one carrying no identifier, which is the note the picture has to draw.
 */
const drawing = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const map = createNodeIdMap()
  return { plex: asPlex(neighbourhood, map.getNodeId), note: map.getNodePath }
}

/**
 * Every edge, as the notes at its two ends. A ticket no note holds reads as
 * nothing, so an edge drawn under anything else says so here.
 */
const edgesOf = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = drawing(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}${edge.label ? ` (${edge.label})` : ''}`,
  )
}

describe('what the plex is handed', () => {
  it('runs an edge the way the relationship runs', () => {
    expect(
      edgesOf(
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
    const [line] = drawing(createNeighbourhood('Here', [['Below', 'child', '', '']])).plex.edges
    expect(line?.label).toBeUndefined()
  })

  it('hangs a sibling off the parent, not off the focus', () => {
    expect(
      edgesOf(
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
      edgesOf(
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
    expect(edgesOf(createNeighbourhood('Here', [['Beside', 'sibling', '', 'Missing']]))).toEqual([])
  })

  it('seats every note it was given', () => {
    const { plex, note } = drawing(
      createNeighbourhood('Here', [
        ['Above', 'parent', 'part of', ''],
        ['Below', 'child', '', ''],
        ['Beside', 'sibling', '', 'Above'],
      ]),
    )
    expect(plex.nodes.map((node) => `${node.seat} ${note(node.id)}`)).toEqual([
      'focus Here',
      'parent Above',
      'child Below',
      'sibling Beside',
    ])
  })
})

describe('what a note is drawn under', () => {
  const held = createNeighbourhood('Here', [
    ['Above', 'parent', '', ''],
    ['Below', 'child', '', ''],
    ['Beside', 'sibling', '', 'Above'],
  ])

  it('is the ticket the note holds, and never the file it is filed under', () => {
    const map = createNodeIdMap()
    const plex = asPlex(held, (path) => `#${path}`)

    expect(plex.nodes.map((node) => node.id)).toEqual(['#Here', '#Above', '#Below', '#Beside'])
    expect(map.getNodePath('Here')).toBe('')
  })

  it('calls every note of one picture something different from every other', () => {
    const { plex } = drawing(held)

    expect(new Set(plex.nodes.map((node) => node.id)).size).toBe(plex.nodes.length)
  })

  it('runs an edge between the tickets of the two notes it joins', () => {
    const plex = asPlex(held, (path) => `#${path}`)

    expect(plex.edges.map((edge) => `${edge.from} -> ${edge.to}`)).toEqual([
      '#Above -> #Here',
      '#Here -> #Below',
      '#Above -> #Beside',
    ])
  })

  it('mints nothing for a parent the picture does not show', () => {
    // A sibling whose parent is off the screen hangs off nothing, and the note
    // it named is one this picture never drew.
    const asked: string[] = []
    asPlex(createNeighbourhood('Here', [['Beside', 'sibling', '', 'Missing']]), (path) => {
      asked.push(path)
      return `#${path}`
    })

    expect(asked).not.toContain('Missing')
  })
})

/** Every edge as the end its arrow is drawn at, and the pair it joins. */
const arrows = (neighbourhood: ReturnType<typeof createNeighbourhood>) => {
  const { plex, note } = drawing(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}: ${edge.arrow ?? 'no arrow'}`,
  )
}

describe('whose wording is on the line', () => {
  it('points the arrow out of the focus, down a line into a child', () => {
    // Marrowfield allotments seats Alice Fenn below it and writes a word on the
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

  it('draws a directional arrow for single child and parent relationships by default', () => {
    // Single child link points down into the child
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Child note', 'child', 'subtask', '', false],
        ]),
      ),
    ).toEqual(['Alice Fenn, The Chair -> Child note: to'])

    // Single parent link points down into the child
    expect(
      arrows(
        createNeighbourhood('Alice Fenn, The Chair', [
          ['Parent note', 'parent', 'depends on', '', false],
        ]),
      ),
    ).toEqual(['Parent note -> Alice Fenn, The Chair: to'])
  })

  it('respects explicit undirected, forward, and reverse direction modes', () => {
    const nh = createNeighbourhood('Alice Fenn, The Chair', [
      ['Child note', 'child', 'subtask', '', false],
    ])
    const map = createNodeIdMap()

    // 1. Explicit undirected mode removes the arrow from a child link
    const undirectedMap = new Map([
      [['Alice Fenn, The Chair', 'Child note'].sort().join(' '), 'undirected'],
    ])
    const undirectedPlex = asPlex(nh, map.getNodeId, undirectedMap)
    expect(undirectedPlex.edges[0]?.arrow).toBeUndefined()

    // 2. Explicit forward mode (Alice -> Child) sets arrow to 'to'
    const forwardMap = new Map([
      [
        ['Alice Fenn, The Chair', 'Child note'].sort().join(' '),
        'Alice Fenn, The Chair->Child note',
      ],
    ])
    const forwardPlex = asPlex(nh, map.getNodeId, forwardMap)
    expect(forwardPlex.edges[0]?.arrow).toBe('to')

    // 3. Explicit reverse mode (Child -> Alice) sets arrow to 'from'
    const reverseMap = new Map([
      [
        ['Alice Fenn, The Chair', 'Child note'].sort().join(' '),
        'Child note->Alice Fenn, The Chair',
      ],
    ])
    const reversePlex = asPlex(nh, map.getNodeId, reverseMap)
    expect(reversePlex.edges[0]?.arrow).toBe('from')
  })

  it('renders consistent arrow directions between two notes regardless of which note is in focus', () => {
    const map = createNodeIdMap()
    const pairKey = ['Parent.md', 'Child.md'].sort().join(' ')

    // Forward direction: Parent -> Child
    const forwardMap = new Map([[pairKey, 'Parent.md->Child.md']])

    // From Parent's perspective (Child is child seat)
    const nhFromParent = createNeighbourhood('Parent.md', [['Child.md', 'child', '', '', false]])
    const plexFromParent = asPlex(nhFromParent, map.getNodeId, forwardMap)
    expect(plexFromParent.edges[0]?.from).toBe(map.getNodeId('Parent.md'))
    expect(plexFromParent.edges[0]?.to).toBe(map.getNodeId('Child.md'))
    expect(plexFromParent.edges[0]?.arrow).toBe('to') // points into Child.md (downwards)

    // From Child's perspective (Parent is parent seat)
    const nhFromChild = createNeighbourhood('Child.md', [['Parent.md', 'parent', '', '', false]])
    const plexFromChild = asPlex(nhFromChild, map.getNodeId, forwardMap)
    expect(plexFromChild.edges[0]?.from).toBe(map.getNodeId('Parent.md'))
    expect(plexFromChild.edges[0]?.to).toBe(map.getNodeId('Child.md'))
    expect(plexFromChild.edges[0]?.arrow).toBe('to') // points into Child.md (downwards)

    // Reverse direction: Child -> Parent
    const reverseMap = new Map([[pairKey, 'Child.md->Parent.md']])

    // From Parent's perspective
    const plexRevFromParent = asPlex(nhFromParent, map.getNodeId, reverseMap)
    expect(plexRevFromParent.edges[0]?.arrow).toBe('from') // points into Parent.md (upwards)

    // From Child's perspective
    const plexRevFromChild = asPlex(nhFromChild, map.getNodeId, reverseMap)
    expect(plexRevFromChild.edges[0]?.arrow).toBe('from') // points into Parent.md (upwards)

    // Undirected: Parent — Child
    const undirectedMap = new Map([[pairKey, 'undirected']])
    const plexUndirectedParent = asPlex(nhFromParent, map.getNodeId, undirectedMap)
    expect(plexUndirectedParent.edges[0]?.arrow).toBeUndefined()
    const plexUndirectedChild = asPlex(nhFromChild, map.getNodeId, undirectedMap)
    expect(plexUndirectedChild.edges[0]?.arrow).toBeUndefined()
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
