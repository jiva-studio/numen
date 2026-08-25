import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { NeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import { asPlex } from './picture'

/** A note and what sits around it: seat, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type Related = [string, Seat, string, string, boolean?]

const around = (focus: string, related: Related[]) =>
  create(NeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: related.map(([path, seat, label, through, answered]) => ({
      note: { path, title: path, identifier: '' },
      seat,
      label,
      through,
      answered: answered ?? false,
    })),
  })

const drawn = (neighbourhood: ReturnType<typeof around>) =>
  asPlex(neighbourhood).edges.map(
    (edge) => `${edge.from} -> ${edge.to}${edge.label ? ` (${edge.label})` : ''}`,
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

  it('writes nothing on a line the person wrote nothing on', () => {
    // A word the application chose would be read as one they had written.
    const [line] = asPlex(around('Here', [['Below', Seat.CHILD, '', '']])).edges
    expect(line?.label).toBeUndefined()
  })

  it('hangs a sibling off the parent, not off the focus', () => {
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
      'Machine learning -> Here',
      'Eigenvector -> Here (needs)',
      'Machine learning -> Clustering',
      'Eigenvector -> SVD',
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
    expect(plex.nodes.map((node) => `${node.seat} ${node.id}`)).toEqual([
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

/** Every edge as the end its arrow is drawn at, and the pair it joins. */
const arrows = (neighbourhood: ReturnType<typeof around>) =>
  asPlex(neighbourhood).edges.map(
    (edge) => `${edge.from} -> ${edge.to}: ${edge.arrow ?? 'no arrow'}`,
  )

describe('whose wording is on the line', () => {
  it('points the arrow out of the focus, down a line into a child', () => {
    // Mahabharata seats Duryodhana below it and writes a word on the link;
    // Duryodhana names the same relationship back.
    expect(
      arrows(
        around('Mahabharata', [['Duryodhana, The King', Seat.CHILD, 'Жлоб', '', true]]),
      ),
    ).toEqual(['Mahabharata -> Duryodhana, The King: to'])
  })

  it('points it out of the focus down a line the relationship runs into', () => {
    // The same link from the other end, where Duryodhana wrote nothing on it.
    // An arrow says who chose the wording, whether or not there is any.
    expect(
      arrows(around('Duryodhana, The King', [['Mahabharata', Seat.PARENT, '', '', true]])),
    ).toEqual(['Mahabharata -> Duryodhana, The King: from'])
  })

  it('points it out of the focus on a jump each note wrote its own word on', () => {
    expect(
      arrows(
        around('Duryodhana, The King', [
          ['Bhishma', Seat.JUMP, 'питамаха, дед рода', '', true],
        ]),
      ),
    ).toEqual(['Bhishma -> Duryodhana, The King: from'])
  })

  it('draws no arrow where only one note named the relationship', () => {
    // Duryodhana jumps to Ebanko and Ebanko says nothing back: one wording,
    // and nothing to choose between.
    expect(
      arrows(
        around('Duryodhana, The King', [
          ['Ebanko', Seat.JUMP, 'яд, поджог, засада', '', false],
        ]),
      ),
    ).toEqual(['Ebanko -> Duryodhana, The King: no arrow'])
  })

  it('draws no arrow on a sibling, whose line is between two other notes', () => {
    // The line hangs off the shared parent, so neither end of it is the note
    // in focus and there is no end for an arrow to name.
    expect(
      arrows(
        around('Duryodhana, The King', [
          ['Mahabharata', Seat.PARENT, '', '', true],
          ['The question of Draupadi', Seat.SIBLING, '', 'Mahabharata', true],
        ]),
      ),
    ).toEqual([
      'Mahabharata -> Duryodhana, The King: from',
      'Mahabharata -> The question of Draupadi: no arrow',
    ])
  })
})
