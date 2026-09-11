/**
 * What the picture calls each note it draws.
 *
 * The failure to watch for is silent: a note handed a second ticket is drawn
 * again from nothing, and a ticket handed to a second note points a gesture at
 * a note the person never touched.
 */
import { describe, expect, it } from 'vitest'
import { createTickets } from './tickets'

describe('a note being drawn', () => {
  it('holds one ticket for as long as the picture draws it', () => {
    const tickets = createTickets()

    expect(tickets.of('Entropy.md')).toBe(tickets.of('Entropy.md'))
  })

  it('holds a ticket no other note in the picture holds', () => {
    const tickets = createTickets()
    const drawn = ['Root.md', 'physics/Entropy.md', 'Heat.md'].map(tickets.of)

    expect(new Set(drawn).size).toBe(3)
  })

  it('holds nothing at all where there is no path to hold it', () => {
    const tickets = createTickets()

    expect(tickets.of('')).toBe('')
  })
})

describe('a note whose file moved', () => {
  it('keeps the ticket it held', () => {
    const tickets = createTickets()
    const ticket = tickets.of('Entropy.md')

    tickets.moved([{ from: 'Entropy.md', to: 'physics/Entropy.md' }])

    expect(tickets.of('physics/Entropy.md')).toBe(ticket)
    expect(tickets.note(ticket)).toBe('physics/Entropy.md')
  })

  it('keeps it through a second move', () => {
    const tickets = createTickets()
    const ticket = tickets.of('Entropy.md')

    tickets.moved([{ from: 'Entropy.md', to: 'physics/Entropy.md' }])
    tickets.moved([{ from: 'physics/Entropy.md', to: 'heat/Entropy.md' }])

    expect(tickets.note(ticket)).toBe('heat/Entropy.md')
  })

  it('is left where it stood by a move of some other note', () => {
    const tickets = createTickets()
    const ticket = tickets.of('Entropy.md')

    tickets.moved([{ from: 'Heat.md', to: 'physics/Heat.md' }])

    expect(tickets.note(ticket)).toBe('Entropy.md')
  })

  it('moves with every other note of a folder that moved at once', () => {
    const tickets = createTickets()
    const one = tickets.of('physics/Entropy.md')
    const two = tickets.of('physics/Heat.md')

    tickets.moved([
      { from: 'physics/Entropy.md', to: 'science/physics/Entropy.md' },
      { from: 'physics/Heat.md', to: 'science/physics/Heat.md' },
    ])

    expect(tickets.note(one)).toBe('science/physics/Entropy.md')
    expect(tickets.note(two)).toBe('science/physics/Heat.md')
  })

  it('trades paths with a note that took its own', () => {
    // Read one move at a time, the first note is moved onto the second and the
    // second is moved back onto the first, and one ticket names both.
    const tickets = createTickets()
    const one = tickets.of('One.md')
    const two = tickets.of('Two.md')

    tickets.moved([
      { from: 'One.md', to: 'Two.md' },
      { from: 'Two.md', to: 'One.md' },
    ])

    expect(tickets.note(one)).toBe('Two.md')
    expect(tickets.note(two)).toBe('One.md')
  })
})

describe('a ticket handed back', () => {
  it('names the note holding it', () => {
    const tickets = createTickets()

    expect(tickets.note(tickets.of('Entropy.md'))).toBe('Entropy.md')
  })

  it('names nothing where no note holds it', () => {
    const tickets = createTickets()
    tickets.of('Entropy.md')

    expect(tickets.note('999')).toBe('')
    expect(tickets.note('Entropy.md')).toBe('')
    expect(tickets.note('')).toBe('')
  })
})

describe('the notes one picture drew', () => {
  it('are kept, and every other note is let go of', () => {
    const tickets = createTickets()
    const kept = tickets.of('Root.md')
    const gone = tickets.of('Heat.md')

    tickets.keeps([kept])

    expect(tickets.note(kept)).toBe('Root.md')
    expect(tickets.note(gone)).toBe('')
  })

  it('leave a note that comes back holding a ticket of its own', () => {
    const tickets = createTickets()
    const root = tickets.of('Root.md')
    const before = tickets.of('Heat.md')
    tickets.keeps([root])

    const after = tickets.of('Heat.md')

    expect(after).not.toBe(before)
    expect(after).not.toBe(root)
    expect(tickets.note(root)).toBe('Root.md')
  })

  it('are all a person travelling from note to note leaves held', () => {
    const tickets = createTickets()
    /** Every ticket minted along the way, whatever became of it. */
    const minted: string[] = []

    for (let step = 0; step < 50; step++) {
      const focus = tickets.of(`Note${step}.md`)
      const beside = tickets.of(`Beside${step}.md`)
      tickets.keeps([focus, beside])
      minted.push(focus, beside)
    }

    expect(minted.filter((ticket) => tickets.note(ticket))).toHaveLength(2)
  })
})
