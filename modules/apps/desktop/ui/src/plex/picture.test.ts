import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { NeighbourhoodResponseSchema, Seat } from '@numen/protocol'
import { asPlex } from './picture'
import { ticketing } from './tickets'

/** A note and what sits around it: seat, label, the note it comes through, and
 *  whether the other note names the relationship too. */
type Related = [string, Seat, string, string, boolean?]

const around = (focus: string, related: Related[]) =>
  create(NeighbourhoodResponseSchema, {
    focus: { path: focus, title: focus, identifier: '' },
    related: related.map(([path, seat, label, through, mutual]) => ({
      note: { path, title: path, identifier: '' },
      seat,
      label,
      through,
      mutual: mutual ?? false,
    })),
  })

/**
 * The picture, and the note each ticket in it stands for. Every note here is
 * one carrying no identifier, which is the note the picture has to draw.
 */
const drawing = (neighbourhood: ReturnType<typeof around>) => {
  const tickets = ticketing()
  return { plex: asPlex(neighbourhood, tickets.of), note: tickets.note }
}

/**
 * Every edge, as the notes at its two ends. A ticket no note holds reads as
 * nothing, so an edge drawn under anything else says so here.
 */
const drawn = (neighbourhood: ReturnType<typeof around>) => {
  const { plex, note } = drawing(neighbourhood)
  return plex.edges.map(
    (edge) => `${note(edge.from)} -> ${note(edge.to)}${edge.label ? ` (${edge.label})` : ''}`,
  )
}

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
    const [line] = drawing(around('Here', [['Below', Seat.CHILD, '', '']])).plex.edges
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
    const { plex, note } = drawing(
      around('Here', [
        ['Above', Seat.PARENT, 'part of', ''],
        ['Below', Seat.CHILD, '', ''],
        ['Beside', Seat.SIBLING, '', 'Above'],
      ]),
    )
    expect(plex.nodes.map((node) => `${node.seat} ${note(node.id)}`)).toEqual([
      'focus Here',
      'parent Above',
      'child Below',
      'sibling Beside',
    ])
  })

  it('leaves out a seat it does not understand', () => {
    expect(drawing(around('Here', [['Odd', Seat.UNSPECIFIED, '', '']])).plex.nodes).toHaveLength(1)
  })
})

describe('what a note is drawn under', () => {
  const held = around('Here', [
    ['Above', Seat.PARENT, '', ''],
    ['Below', Seat.CHILD, '', ''],
    ['Beside', Seat.SIBLING, '', 'Above'],
  ])

  it('is the ticket the note holds, and never the file it is filed under', () => {
    const tickets = ticketing()
    const plex = asPlex(held, (path) => `#${path}`)

    expect(plex.nodes.map((node) => node.id)).toEqual(['#Here', '#Above', '#Below', '#Beside'])
    expect(tickets.note('Here')).toBe('')
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
    asPlex(around('Here', [['Beside', Seat.SIBLING, '', 'Missing']]), (path) => {
      asked.push(path)
      return `#${path}`
    })

    expect(asked).not.toContain('Missing')
  })
})

/** Every edge as the end its arrow is drawn at, and the pair it joins. */
const arrows = (neighbourhood: ReturnType<typeof around>) => {
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
        around('Marrowfield allotments', [
          ['Alice Fenn, The Chair', Seat.CHILD, 'the chair since May', '', true],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: to'])
  })

  it('points it out of the focus down a line the relationship runs into', () => {
    // The same link from the other end, where Alice Fenn wrote nothing on it.
    // An arrow says who chose the wording, whether or not there is any.
    expect(
      arrows(
        around('Alice Fenn, The Chair', [
          ['Marrowfield allotments', Seat.PARENT, '', '', true],
        ]),
      ),
    ).toEqual(['Marrowfield allotments -> Alice Fenn, The Chair: from'])
  })

  it('points it out of the focus on a jump each note wrote its own word on', () => {
    expect(
      arrows(
        around('Alice Fenn, The Chair', [
          ['Bram Doyle', Seat.JUMP, 'the oldest tenant', '', true],
        ]),
      ),
    ).toEqual(['Bram Doyle -> Alice Fenn, The Chair: from'])
  })

  it('draws no arrow where only one note named the relationship', () => {
    // Alice Fenn jumps to Cora Hale and Cora Hale says nothing back: one
    // wording, and nothing to choose between.
    expect(
      arrows(
        around('Alice Fenn, The Chair', [
          ['Cora Hale', Seat.JUMP, 'keys, hoses, the gate', '', false],
        ]),
      ),
    ).toEqual(['Cora Hale -> Alice Fenn, The Chair: no arrow'])
  })

  it('draws no arrow on a sibling, whose line is between two other notes', () => {
    // The line hangs off the shared parent, so neither end of it is the note
    // in focus and there is no end for an arrow to name.
    expect(
      arrows(
        around('Alice Fenn, The Chair', [
          ['Marrowfield allotments', Seat.PARENT, '', '', true],
          ['The question of the rota', Seat.SIBLING, '', 'Marrowfield allotments', true],
        ]),
      ),
    ).toEqual([
      'Marrowfield allotments -> Alice Fenn, The Chair: from',
      'Marrowfield allotments -> The question of the rota: no arrow',
    ])
  })
})
