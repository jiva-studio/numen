import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { NeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import { asPlex } from './plex'

const around = (focus: string, related: [string, Seat][]) =>
  create(NeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: related.map(([path, seat]) => ({
      note: { path, title: path, identifier: '' },
      seat,
    })),
  })

const drawn = (neighbourhood: ReturnType<typeof around>) =>
  asPlex(neighbourhood).edges.map((edge) => `${edge.from} -> ${edge.to}`)

describe('what the plex is handed', () => {
  it('runs an edge the way the relationship runs', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT],
          ['Below', Seat.CHILD],
          ['Across', Seat.JUMP],
        ]),
      ),
    ).toEqual(['Above -> Here', 'Here -> Below', 'Across -> Here'])
  })

  it('hangs a sibling off the parent, not off the focus', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT],
          ['Beside', Seat.SIBLING],
        ]),
      ),
    ).toEqual(['Above -> Here', 'Above -> Beside'])
  })

  it('draws no sibling edge when no parent is shown', () => {
    // It would otherwise fall back to the focus, which says the wrong thing:
    // a sibling is not a child.
    expect(drawn(around('Here', [['Beside', Seat.SIBLING]]))).toEqual([])
  })

  it('seats every note it was given', () => {
    const plex = asPlex(
      around('Here', [
        ['Above', Seat.PARENT],
        ['Below', Seat.CHILD],
        ['Beside', Seat.SIBLING],
      ]),
    )
    expect(plex.nodes.map((node) => `${node.role} ${node.id}`)).toEqual([
      'focus Here',
      'parent Above',
      'child Below',
      'sibling Beside',
    ])
  })

  it('leaves out a seat it does not understand', () => {
    expect(asPlex(around('Here', [['Odd', Seat.UNSPECIFIED]])).nodes).toHaveLength(1)
  })
})
