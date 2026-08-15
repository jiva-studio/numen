import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { NeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import { asPlex } from './plex'

const around = (focus: string, related: [string, Seat, string, string][]) =>
  create(NeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: related.map(([path, seat, label, through]) => ({
      note: { path, title: path, identifier: '' },
      seat,
      label,
      through,
    })),
  })

const drawn = (neighbourhood: ReturnType<typeof around>) =>
  asPlex(neighbourhood).edges.map((edge) => `${edge.from} -> ${edge.to} (${edge.label})`)

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
    ).toEqual([
      'Above -> Here (part of)',
      // Nothing was written on this one, so the seat says what it is.
      'Here -> Below (child)',
      'Across -> Here (see also)',
    ])
  })

  it('hangs a sibling off the parent, not off the focus', () => {
    expect(
      drawn(
        around('Here', [
          ['Above', Seat.PARENT, 'part of', ''],
          ['Beside', Seat.SIBLING, '', 'Above'],
        ]),
      ),
    ).toEqual(['Above -> Here (part of)', 'Above -> Beside (sibling)'])
  })

  it('hangs each sibling off its own parent when there are two', () => {
    // Each sibling belongs to one of the two parents, and only the answer
    // knows which. A line drawn from the other one says a note is the child of
    // something the vault never joined it to.
    expect(
      drawn(
        around('Here', [
          ['Machine learning', Seat.PARENT, '', ''],
          ['Eigenvector', Seat.PARENT, 'needs', ''],
          ['Clustering', Seat.SIBLING, '', 'Machine learning'],
          ['SVD', Seat.SIBLING, '', 'Eigenvector'],
        ]),
      ),
    ).toEqual([
      'Machine learning -> Here (parent)',
      'Eigenvector -> Here (needs)',
      'Machine learning -> Clustering (sibling)',
      'Eigenvector -> SVD (sibling)',
    ])
  })

  it('draws no sibling edge when no parent is shown', () => {
    // It would otherwise fall back to the focus, which says the wrong thing:
    // a sibling is not a child.
    expect(drawn(around('Here', [['Beside', Seat.SIBLING, '', 'Missing']]))).toEqual([])
  })

  it('seats every note it was given', () => {
    const plex = asPlex(
      around('Here', [
        ['Above', Seat.PARENT, 'part of', ''],
        ['Below', Seat.CHILD, '', ''],
        ['Beside', Seat.SIBLING, '', 'Above'],
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
    expect(asPlex(around('Here', [['Odd', Seat.UNSPECIFIED, '', '']])).nodes).toHaveLength(1)
  })
})
