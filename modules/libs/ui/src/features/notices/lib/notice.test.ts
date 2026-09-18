/**
 * What the notices in the corner decide, as plain values.
 */
import { describe, expect, it } from 'vitest'
import { PER_WORD, SETTLE, arrivals, dwellOf, getFinishedNotices, getShownNotices } from './dwell'
import { foldNotices } from './fold'
import { measureMovement, type Movement } from './movement'
import { getStillAway, getReadable, tallyOf, type Notice } from './notice'

const createNotice = (over: Partial<Notice> = {}): Notice => ({
  id: 'embedding',
  says: 'Preparing search by meaning',
  ...over,
})

describe('which notices stand in the corner', () => {
  it('draws the notices that have something to say, in the order given', () => {
    const first = createNotice({ id: 'one' })
    const second = createNotice({ id: 'two', says: 'Reading the vault' })
    expect(getReadable([first, second]).map((each) => each.id)).toEqual(['one', 'two'])
  })

  it('leaves out a notice nobody could read', () => {
    expect(getReadable([createNotice({ id: 'silent', says: '' }), createNotice()]).map((each) => each.id)).toEqual([
      'embedding',
    ])
  })

  it('draws nothing where nothing is running', () => {
    expect(getReadable([])).toEqual([])
  })
})

describe('what a notice counts against', () => {
  it('counts where both halves of the count are known', () => {
    expect(tallyOf(createNotice({ done: 3, total: 8 }))).toEqual({ done: 3, total: 8 })
  })

  it('counts nothing where either half is missing', () => {
    // A pass that has not said what it found has no total to draw against, and
    // a bar drawn against a total nobody gave means nothing.
    expect(tallyOf(createNotice({ done: 3 }))).toBeUndefined()
    expect(tallyOf(createNotice({ total: 8 }))).toBeUndefined()
    expect(tallyOf(createNotice())).toBeUndefined()
  })

  it('counts a total of nothing, and leaves the share to whoever draws it', () => {
    expect(tallyOf(createNotice({ done: 0, total: 0 }))).toEqual({ done: 0, total: 0 })
  })
})

describe('a notice put away', () => {
  it('is not drawn, and the rest are', () => {
    const notices = [createNotice({ id: 'reading' }), createNotice({ id: 'embedding' })]
    const arrived = arrivals(new Map(), notices, 0)
    expect(
      getShownNotices(notices, arrived, new Set(['reading']), 20_000).map((each) => each.id),
    ).toEqual(['embedding'])
  })

  it('stays away while the work it was about is still running', () => {
    const notices = [createNotice({ id: 'embedding' })]
    expect([...getStillAway(new Set(['embedding']), notices)]).toEqual(['embedding'])
  })

  it('is forgotten once that work has ended, so the next one is news', () => {
    expect([...getStillAway(new Set(['embedding']), [])]).toEqual([])
  })
})

describe('how long work runs before it is worth a card', () => {
  const embedding = [createNotice({ id: 'embedding' })]

  it('keeps the moment a notice arrived, and forgets one that has gone', () => {
    const first = arrivals(new Map(), embedding, 1000)
    expect([...first]).toEqual([['embedding', 1000]])

    const later = arrivals(first, embedding, 9000)
    expect(later.get('embedding')).toBe(1000)

    expect([...arrivals(later, [], 9000)]).toEqual([])
  })

  it('draws nothing while the work is younger than the wait', () => {
    const arrived = arrivals(new Map(), embedding, 1000)
    expect(getShownNotices(embedding, arrived, new Set(), 10_999, 10_000)).toEqual([])
  })

  it('draws it once the work has lasted', () => {
    const arrived = arrivals(new Map(), embedding, 1000)
    expect(
      getShownNotices(embedding, arrived, new Set(), 11_000, 10_000).map((each) => each.id),
    ).toEqual(['embedding'])
  })

  it('draws nothing it was never told the arrival of', () => {
    expect(getShownNotices(embedding, new Map(), new Set(), 99_000, 10_000)).toEqual([])
  })
})

describe('a notice somebody asked for', () => {
  const asked: Notice = { id: 'reading', says: 'Reading a scan', isAsked: true }
  const behind: Notice = { id: 'indexing', says: 'Indexing' }

  it('is drawn the moment it arrives', () => {
    // The wait is for work nobody asked for. Somebody who asked is waiting to
    // be told it began, and ten seconds of nothing is an application that did
    // not hear them.
    const arrived = arrivals(new Map(), [asked, behind], 0)

    expect(getShownNotices([asked, behind], arrived, new Set(), 0).map((createNotice) => createNotice.id)).toEqual([
      'reading',
    ])
  })

  it('is put away like any other', () => {
    const arrived = arrivals(new Map(), [asked], 0)

    expect(getShownNotices([asked], arrived, new Set(['reading']), 0)).toEqual([])
  })
})

describe('how long something said stands to be read', () => {
  it('gives more time to more words', () => {
    expect(dwellOf('Renamed')).toBe(SETTLE + PER_WORD)
    expect(dwellOf('Renamed', 'One.md')).toBe(SETTLE + 2 * PER_WORD)
  })

  it('never runs out for a notice too long to be read in passing', () => {
    const list = Array.from({ length: 21 }, (_, at) => `Note${at}.md`).join(' ')

    expect(dwellOf('Links repaired in', list)).toBe(Infinity)
  })

  it('is drawn the moment it arrives and goes once it has been read', () => {
    const said: Notice = { id: 'renamed', says: 'Renamed', stay: 'read' }
    const arrived = arrivals(new Map(), [said], 1000)

    expect(getShownNotices([said], arrived, new Set(), 1000).map((each) => each.id)).toEqual([
      'renamed',
    ])
    expect(getShownNotices([said], arrived, new Set(), 1000 + SETTLE + PER_WORD - 1)).toHaveLength(
      1,
    )
    expect(getShownNotices([said], arrived, new Set(), 1000 + SETTLE + PER_WORD)).toEqual([])
  })

  it('stands until it is put away where it was not asked to be read in passing', () => {
    const kept: Notice = {
      id: 'occupied',
      says: 'A note of that name is filed there',
      stay: 'kept',
    }
    const arrived = arrivals(new Map(), [kept], 0)

    expect(getShownNotices([kept], arrived, new Set(), 10_000_000).map((each) => each.id)).toEqual([
      'occupied',
    ])
  })

  it('keeps a notice too long to be read in passing standing, and never finishes it', () => {
    const list = Array.from({ length: 21 }, (_, at) => `Note${at}.md`).join(' ')
    const long: Notice = { id: 'repaired', says: 'Links repaired in', about: list, stay: 'read' }
    const arrived = arrivals(new Map(), [long], 0)

    expect(getShownNotices([long], arrived, new Set(), 10_000_000).map((each) => each.id)).toEqual([
      'repaired',
    ])
    expect(getFinishedNotices([long], arrived, 10_000_000)).toEqual([])
  })

  it('names the ones whose caller may dismiss them, and only those', () => {
    const said: Notice = { id: 'renamed', says: 'Renamed', stay: 'read' }
    const kept: Notice = { id: 'occupied', says: 'Filed there already', stay: 'kept' }
    const work: Notice = { id: 'embedding', says: 'Indexing', isWorking: true }
    const all = [said, kept, work]
    const arrived = arrivals(new Map(), all, 0)

    expect(getFinishedNotices(all, arrived, 0)).toEqual([])
    expect(getFinishedNotices(all, arrived, 1_000_000)).toEqual(['renamed'])
  })
})

describe('how many cards stand at once', () => {
  const work = (id: string): Notice => ({ id, says: 'Indexing', isWorking: true })
  const word = (id: string): Notice => ({ id, says: 'Renamed', stay: 'read' })

  it('folds nothing while there is room', () => {
    expect(foldNotices([work('a'), word('b')], 4)).toEqual({
      shown: [work('a'), word('b')],
      over: 0,
    })
  })

  it('folds the oldest of what has been said, and counts them', () => {
    const drawn = [word('a'), word('b'), work('c'), word('d')]

    expect(foldNotices(drawn, 2)).toEqual({ shown: [work('c'), word('d')], over: 2 })
  })

  it('folds no work and nothing that is so, however many there are', () => {
    const drawn = [work('a'), work('b'), work('c'), work('d'), work('e')]

    expect(foldNotices(drawn, 2)).toEqual({ shown: drawn, over: 0 })
  })

  it('keeps the order of what it did not fold', () => {
    const drawn = [word('a'), word('b'), word('c')]

    expect(foldNotices(drawn, 2).shown.map((each) => each.id)).toEqual(['b', 'c'])
  })

  it('folds nothing that stopped badly, wherever it stands', () => {
    // Trouble is what a person has to see, and it arrives before the reports
    // that pile up behind it.
    const failed: Notice = { id: 'failed', says: 'Reading', tone: 'alarm', stay: 'kept' }
    const drawn = [failed, word('a'), word('b'), word('c')]

    const { shown, over } = foldNotices(drawn, 2)

    expect(shown.map((each) => each.id)).toEqual(['failed', 'c'])
    expect(over).toBe(2)
  })
})

describe('measuring how fast a count moves', () => {
  const createFetching = (count: number): Notice => ({
    id: 'model',
    says: 'Preparing the model',
    done: count,
    total: 470_268_510,
    counting: 'bytes',
    isWorking: true,
  })

  it('reads a count once before it has a rate', () => {
    const moving = measureMovement(new Map(), [createFetching(0)], 1000)

    expect(moving.get('model')).toEqual({ done: 0, rate: 0, at: 1000 })
  })

  it('measures how fast the count is moving between two readings', () => {
    const first = measureMovement(new Map(), [createFetching(0)], 1000)
    const second = measureMovement(first, [createFetching(4_000_000)], 3000)

    expect(second.get('model')?.rate).toBe(2_000_000)
  })

  it('measures a group over the stretch it took, not the moment it landed in', () => {
    // Ten seconds of readings, four million arriving at the end of them. The
    // work did four hundred thousand a second, whatever the last reading saw.
    let moving = measureMovement(new Map(), [createFetching(0)], 1000)
    for (let at = 1250; at < 11_000; at += 250) {
      moving = measureMovement(moving, [createFetching(0)], at)
    }
    moving = measureMovement(moving, [createFetching(4_000_000)], 11_000)

    expect(moving.get('model')?.rate).toBe(400_000)
  })

  it('forgets work that is no longer standing', () => {
    const was: ReadonlyMap<string, Movement> = new Map([['gone', { done: 5, rate: 1, at: 0 }]])

    expect(measureMovement(was, [createFetching(0)], 1000).has('gone')).toBe(false)
  })

  it('measures nothing for work with no total to count against', () => {
    const nothing: Notice = { id: 'scan', says: 'Reading a scan', isWorking: true }

    expect(measureMovement(new Map(), [nothing], 1000).size).toBe(0)
  })
})
