/**
 * What the phone hands the plex: which nodes are drawn, which way each edge
 * runs, and which end of it carries an arrow.
 */
import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { GetNeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import { asPlex } from './picture'

/** A note and what sits around it: seat, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type Neighbour = [string, Seat, string, string, boolean?]

const around = (focus: string, related: Neighbour[]) =>
  create(GetNeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: related.map(([path, seat, label, through, mutual]) => ({
      note: { path, title: path, identifier: '' },
      seat,
      label,
      through,
      mutual: mutual ?? false,
    })),
  })

/** Every edge, as the notes at its two ends and what is written on it. */
const drawn = (neighbourhood: ReturnType<typeof around>) =>
  asPlex(neighbourhood).edges.map(
    (edge) =>
      `${edge.from} -> ${edge.to}${edge.label ? ` (${edge.label})` : ''}` +
      `${edge.arrow ? ` [${edge.arrow}]` : ''}`,
  )

describe('what the plex is handed', () => {
  it('runs an edge the way the relationship runs', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT, 'part of', ''],
          ['Below', Seat.CHILD, '', ''],
          ['Across', Seat.JUMP, 'see also', ''],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Here -> Below', 'Across -> Here (see also)'])
  })

  it('draws the focus and everything seated around it, under the path each holds', () => {
    const { nodes } = asPlex(
      around('Here', [
        ['Above', Seat.PARENT, '', ''],
        ['Below', Seat.CHILD, '', ''],
      ]),
    )
    expect(nodes.map((node) => [node.id, node.seat])).toEqual([
      ['Here', 'focus'],
      ['Above', 'parent'],
      ['Below', 'child'],
    ])
  })

  it('writes nothing on a line the person wrote nothing on', () => {
    const [line] = asPlex(around('Here', [['Below', Seat.CHILD, '', '']])).edges
    expect(line?.label).toBeUndefined()
  })

  // The arrow marks the far end of a relationship both notes named, so it
  // points away from the focus whichever way the edge itself runs.
  it('marks the end away from the focus where both notes named the link', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT, '', '', true],
          ['Below', Seat.CHILD, '', '', true],
        ]),
      ),
    ).toEqual(['Above -> Here [from]', 'Here -> Below [to]'])
  })

  it('leaves a one-sided link without an arrow', () => {
    expect(drawn(around('Here', [['Below', Seat.CHILD, '', '']]))).toEqual(['Here -> Below'])
  })

  it('hangs a sibling off the parent the two share, not off the focus', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT, 'part of', ''],
          ['Beside', Seat.SIBLING, '', 'Above'],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Above -> Beside'])
  })

  it('hangs each sibling off its own parent when there are two', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT, '', ''],
          ['Beyond', Seat.PARENT, '', ''],
          ['Beside', Seat.SIBLING, '', 'Beyond'],
        ]),
      ),
    ).toEqual(['Above -> Here', 'Beyond -> Here', 'Beyond -> Beside'])
  })

  // A line to a note nobody drew runs off the picture.
  it('drops a sibling whose parent is not drawn', () => {
    const { nodes, edges } = asPlex(around('Here', [['Beside', Seat.SIBLING, '', 'Elsewhere']]))
    expect(nodes.map((node) => node.id)).toEqual(['Here', 'Beside'])
    expect(edges).toEqual([])
  })

  it('draws nothing for a seat it does not know', () => {
    const { nodes, edges } = asPlex(around('Here', [['Below', Seat.UNSPECIFIED, '', '']]))
    expect(nodes.map((node) => node.id)).toEqual(['Here'])
    expect(edges).toEqual([])
  })
})
