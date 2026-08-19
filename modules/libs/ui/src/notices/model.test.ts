/**
 * What the notices in the corner decide, as plain values.
 */
import { describe, expect, it } from 'vitest'
import { arrivals, remembered, showing, standing, tallyOf, type Notice } from './model'

const one = (over: Partial<Notice> = {}): Notice => ({
  id: 'embedding',
  says: 'Preparing search by meaning',
  ...over,
})

describe('which notices stand in the corner', () => {
  it('draws the notices that have something to say, in the order given', () => {
    const first = one({ id: 'one' })
    const second = one({ id: 'two', says: 'Reading the vault' })
    expect(standing([first, second]).map((each) => each.id)).toEqual(['one', 'two'])
  })

  it('leaves out a notice nobody could read', () => {
    expect(standing([one({ id: 'silent', says: '' }), one()]).map((each) => each.id)).toEqual([
      'embedding',
    ])
  })

  it('draws nothing where nothing is running', () => {
    expect(standing([])).toEqual([])
  })
})

describe('what a notice counts against', () => {
  it('counts where both halves of the count are known', () => {
    expect(tallyOf(one({ done: 3, total: 8 }))).toEqual({ done: 3, total: 8 })
  })

  it('counts nothing where either half is missing', () => {
    // A pass that has not said what it found has no total to draw against, and
    // a bar drawn against a total nobody gave means nothing.
    expect(tallyOf(one({ done: 3 }))).toBeUndefined()
    expect(tallyOf(one({ total: 8 }))).toBeUndefined()
    expect(tallyOf(one())).toBeUndefined()
  })

  it('counts a total of nothing, and leaves the share to whoever draws it', () => {
    expect(tallyOf(one({ done: 0, total: 0 }))).toEqual({ done: 0, total: 0 })
  })
})

describe('a notice put away', () => {
  it('is not drawn, and the rest are', () => {
    const notices = [one({ id: 'reading' }), one({ id: 'embedding' })]
    const arrived = arrivals(new Map(), notices, 0)
    expect(
      showing(notices, arrived, new Set(['reading']), 20_000).map((each) => each.id),
    ).toEqual(['embedding'])
  })

  it('stays away while the work it was about is still running', () => {
    const notices = [one({ id: 'embedding' })]
    expect([...remembered(new Set(['embedding']), notices)]).toEqual(['embedding'])
  })

  it('is forgotten once that work has ended, so the next one is news', () => {
    expect([...remembered(new Set(['embedding']), [])]).toEqual([])
  })
})

describe('how long work runs before it is worth a card', () => {
  const embedding = [one({ id: 'embedding' })]

  it('keeps the moment a notice arrived, and forgets one that has gone', () => {
    const first = arrivals(new Map(), embedding, 1000)
    expect([...first]).toEqual([['embedding', 1000]])

    const later = arrivals(first, embedding, 9000)
    expect(later.get('embedding')).toBe(1000)

    expect([...arrivals(later, [], 9000)]).toEqual([])
  })

  it('draws nothing while the work is younger than the wait', () => {
    const arrived = arrivals(new Map(), embedding, 1000)
    expect(showing(embedding, arrived, new Set(), 10_999, 10_000)).toEqual([])
  })

  it('draws it once the work has lasted', () => {
    const arrived = arrivals(new Map(), embedding, 1000)
    expect(showing(embedding, arrived, new Set(), 11_000, 10_000).map((each) => each.id)).toEqual([
      'embedding',
    ])
  })

  it('draws nothing it was never told the arrival of', () => {
    expect(showing(embedding, new Map(), new Set(), 99_000, 10_000)).toEqual([])
  })
})
