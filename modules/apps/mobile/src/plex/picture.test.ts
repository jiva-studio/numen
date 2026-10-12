/**
 * What the phone hands the plex: which nodes are drawn, which way each edge
 * runs, and which end of it carries an arrow.
 */
import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { GetNeighbourhoodResponseSchema, Relation } from '@numen/protocol'
import { asPlex } from './picture'

/** A note and what sits around it: seat, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type Neighbour = [string, Relation, string, string, boolean?]

const createNeighbourhood = (focus: string, neighbours: Neighbour[]) =>
  create(GetNeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: neighbours.map(([path, seat, label, through, mutual]) => ({
      note: { path, title: path, identifier: '' },
      relation: seat,
      label,
      through,
      isMutual: mutual ?? false,
    })),
  })

/** Every edge, as the notes at its two ends and what is written on it. */
const getEdges = (neighbourhood: ReturnType<typeof createNeighbourhood>) =>
  asPlex(neighbourhood).edges.map(
    (edge) =>
      `${edge.from} -> ${edge.to}${edge.label ? ` (${edge.label})` : ''}` +
      `${edge.arrow ? ` [${edge.arrow}]` : ''}`,
  )

describe('what the plex is handed', () => {
  it('runs an edge the way the relationship runs', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', Relation.PARENT, 'part of', ''],
          ['Below', Relation.CHILD, '', ''],
          ['Across', Relation.JUMP, 'see also', ''],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Here -> Below', 'Across -> Here (see also)'])
  })

  it('draws the focus and everything seated around it, under the path each holds', () => {
    const { nodes } = asPlex(
      createNeighbourhood('Here', [
        ['Above', Relation.PARENT, '', ''],
        ['Below', Relation.CHILD, '', ''],
      ]),
    )
    expect(nodes.map((node) => [node.id, node.seat])).toEqual([
      ['Here', 'focus'],
      ['Above', 'parent'],
      ['Below', 'child'],
    ])
  })

  it('writes nothing on a line the person wrote nothing on', () => {
    const [line] = asPlex(createNeighbourhood('Here', [['Below', Relation.CHILD, '', '']])).edges
    expect(line?.label).toBeUndefined()
  })

  // The arrow marks the far end of a relationship both notes named, so it
  // points away from the focus whichever way the edge itself runs.
  it('marks the end away from the focus where both notes named the link', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', Relation.PARENT, '', '', true],
          ['Below', Relation.CHILD, '', '', true],
        ]),
      ),
    ).toEqual(['Above -> Here [from]', 'Here -> Below [to]'])
  })

  it('leaves a one-sided link without an arrow', () => {
    expect(getEdges(createNeighbourhood('Here', [['Below', Relation.CHILD, '', '']]))).toEqual([
      'Here -> Below',
    ])
  })

  it('hangs a sibling off the parent the two share, not off the focus', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', Relation.PARENT, 'part of', ''],
          ['Beside', Relation.SIBLING, '', 'Above'],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Above -> Beside'])
  })

  it('hangs each sibling off its own parent when there are two', () => {
    expect(
      getEdges(
        createNeighbourhood('Here', [
          ['Above', Relation.PARENT, '', ''],
          ['Beyond', Relation.PARENT, '', ''],
          ['Beside', Relation.SIBLING, '', 'Beyond'],
        ]),
      ),
    ).toEqual(['Above -> Here', 'Beyond -> Here', 'Beyond -> Beside'])
  })

  // A line to a note nobody drew runs off the picture.
  it('drops a sibling whose parent is not drawn', () => {
    const { nodes, edges } = asPlex(
      createNeighbourhood('Here', [['Beside', Relation.SIBLING, '', 'Elsewhere']]),
    )
    expect(nodes.map((node) => node.id)).toEqual(['Here', 'Beside'])
    expect(edges).toEqual([])
  })

  it('draws nothing for a seat it does not know', () => {
    const { nodes, edges } = asPlex(
      createNeighbourhood('Here', [['Below', Relation.UNSPECIFIED, '', '']]),
    )
    expect(nodes.map((node) => node.id)).toEqual(['Here'])
    expect(edges).toEqual([])
  })

  it('hangs a sibling off the focus when the focus is the parent they share', () => {
    expect(
      getEdges(createNeighbourhood('Here', [['Beside', Relation.SIBLING, 'next to', 'Here']])),
    ).toEqual(['Here -> Beside (next to)'])
  })

  it('draws only the focus, and no edge, around a note with nothing seated near it', () => {
    const { nodes, edges } = asPlex(createNeighbourhood('Here', []))
    expect(nodes).toEqual([{ id: 'Here', title: 'Here', seat: 'focus' }])
    expect(edges).toEqual([])
  })

  it('draws the focus under an empty path and title when the neighbourhood has none', () => {
    const { nodes, edges } = asPlex(create(GetNeighbourhoodResponseSchema, {}))
    expect(nodes).toEqual([{ id: '', title: '', seat: 'focus' }])
    expect(edges).toEqual([])
  })

  it('titles the focus with the title its note holds', () => {
    const { nodes } = asPlex(
      create(GetNeighbourhoodResponseSchema, {
        focus: { path: 'here.md', title: 'Here', identifier: '' },
      }),
    )
    expect(nodes[0]).toEqual({ id: 'here.md', title: 'Here', seat: 'focus' })
  })
})
