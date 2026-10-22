/**
 * What the notices in the corner decide, as plain values.
 */
import { describe, expect, it } from 'vitest'
import {
  PER_WORD,
  MIN_DURATION,
  getArrivalTimes,
  getNoticeDuration,
  getFinishedNotices,
  getShownNotices,
} from './duration'
import { foldNotices } from './fold'
import { measureMovement, type Movement } from './movement'
import { getStillAway, getReadable, findTally, type Notice } from './notice'

const createNotice = (overrides: Partial<Notice> = {}): Notice => ({
  id: 'embedding',
  text: 'Preparing search by meaning',
  ...overrides,
})

describe('which notices stand in the corner', () => {
  it('draws the notices that have something to say, in the order given', () => {
    const first = createNotice({ id: 'one' })
    const second = createNotice({ id: 'two', text: 'Reading the vault' })
    expect(getReadable([first, second]).map((each) => each.id)).toEqual(['one', 'two'])
  })

  it('leaves out a notice nobody could read', () => {
    expect(
      getReadable([createNotice({ id: 'silent', text: '' }), createNotice()]).map(
        (each) => each.id,
      ),
    ).toEqual(['embedding'])
  })

  it('draws nothing where nothing is running', () => {
    expect(getReadable([])).toEqual([])
  })
})

describe('what a notice counts against', () => {
  it('counts where both halves of the count are known', () => {
    expect(findTally(createNotice({ done: 3, total: 8 }))).toEqual({ completedCount: 3, total: 8 })
  })

  it('counts nothing where either half is missing', () => {
    // A pass that has not said what it found has no total to draw against, and
    // a bar drawn against a total nobody gave means nothing.
    expect(findTally(createNotice({ done: 3 }))).toBeUndefined()
    expect(findTally(createNotice({ total: 8 }))).toBeUndefined()
    expect(findTally(createNotice())).toBeUndefined()
  })

  it('counts a total of nothing, and leaves the share to whoever draws it', () => {
    expect(findTally(createNotice({ done: 0, total: 0 }))).toEqual({ completedCount: 0, total: 0 })
  })
})

describe('a notice put away', () => {
  it('is not drawn, and the rest are', () => {
    const notices = [createNotice({ id: 'reading' }), createNotice({ id: 'embedding' })]
    const arrivalTimes = getArrivalTimes(new Map(), notices, 0)
    expect(
      getShownNotices(notices, 20_000, {
        firstSeen: arrivalTimes,
        away: new Set(['reading']),
      }).map((each) => each.id),
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
    const first = getArrivalTimes(new Map(), embedding, 1000)
    expect([...first]).toEqual([['embedding', 1000]])

    const later = getArrivalTimes(first, embedding, 9000)
    expect(later.get('embedding')).toBe(1000)

    expect([...getArrivalTimes(later, [], 9000)]).toEqual([])
  })

  it('suppresses display until debounce delay expires', () => {
    const arrivalTimes = getArrivalTimes(new Map(), embedding, 1000)
    expect(getShownNotices(embedding, 10_999, { firstSeen: arrivalTimes, wait: 10_000 })).toEqual(
      [],
    )
  })

  it('draws it once the work has lasted', () => {
    const arrivalTimes = getArrivalTimes(new Map(), embedding, 1000)
    expect(
      getShownNotices(embedding, 11_000, { firstSeen: arrivalTimes, wait: 10_000 }).map(
        (each) => each.id,
      ),
    ).toEqual(['embedding'])
  })

  it('draws nothing it was never told the arrival of', () => {
    expect(getShownNotices(embedding, 99_000, { firstSeen: new Map(), wait: 10_000 })).toEqual([])
  })
})

describe('a notice somebody requested for', () => {
  const request: Notice = { id: 'reading', text: 'Recognising a scan', isUserStarted: true }
  const behindNotice: Notice = { id: 'indexing', text: 'Indexing' }

  it('is drawn the moment it arrives', () => {
    // The wait is for work nobody asked for. Somebody who asked is waiting to
    // be told it began, and ten seconds of nothing is an application that did
    // not hear them.
    const arrivalTimes = getArrivalTimes(new Map(), [request, behindNotice], 0)

    expect(
      getShownNotices([request, behindNotice], 0, { firstSeen: arrivalTimes }).map(
        (createNotice) => createNotice.id,
      ),
    ).toEqual(['reading'])
  })

  it('is put away like any other', () => {
    const arrivalTimes = getArrivalTimes(new Map(), [request], 0)

    expect(
      getShownNotices([request], 0, { firstSeen: arrivalTimes, away: new Set(['reading']) }),
    ).toEqual([])
  })
})

describe('how long something returned remains to be read', () => {
  it('gives more time to more words', () => {
    expect(getNoticeDuration('Renamed')).toBe(MIN_DURATION + PER_WORD)
    expect(getNoticeDuration('Renamed', 'One.md')).toBe(MIN_DURATION + 2 * PER_WORD)
  })

  it('never runs out for a notice too long to be read in passing', () => {
    const list = Array.from({ length: 21 }, (_, index) => `Note${index}.md`).join(' ')

    expect(getNoticeDuration('Links repaired in', list)).toBe(Infinity)
  })

  it('remains for as long as twenty words, which is not too much to read in passing', () => {
    const list = Array.from({ length: 20 }, (_, index) => `Note${index}.md`).join(' ')

    expect(getNoticeDuration(list)).toBe(MIN_DURATION + 20 * PER_WORD)
  })

  it('is drawn the moment it arrives and goes once it has been read', () => {
    const saidText: Notice = { id: 'renamed', text: 'Renamed', stay: 'read' }
    const arrivalTimes = getArrivalTimes(new Map(), [saidText], 1000)

    expect(
      getShownNotices([saidText], 1000, { firstSeen: arrivalTimes }).map((each) => each.id),
    ).toEqual(['renamed'])
    expect(
      getShownNotices([saidText], 1000 + MIN_DURATION + PER_WORD - 1, { firstSeen: arrivalTimes }),
    ).toHaveLength(1)
    expect(
      getShownNotices([saidText], 1000 + MIN_DURATION + PER_WORD, { firstSeen: arrivalTimes }),
    ).toEqual([])
  })

  it('remains until it is put away where it was not requested to be read in passing', () => {
    const retainedNotice: Notice = {
      id: 'occupied',
      text: 'A note of that name is filed there',
      stay: 'kept',
    }
    const arrivalTimes = getArrivalTimes(new Map(), [retainedNotice], 0)

    expect(
      getShownNotices([retainedNotice], 10_000_000, { firstSeen: arrivalTimes }).map(
        (each) => each.id,
      ),
    ).toEqual(['occupied'])
  })

  it('keeps a notice too long to be read in passing standing, and never finishes it', () => {
    const list = Array.from({ length: 21 }, (_, index) => `Note${index}.md`).join(' ')
    const long: Notice = { id: 'repaired', text: 'Links repaired in', about: list, stay: 'read' }
    const arrivalTimes = getArrivalTimes(new Map(), [long], 0)

    expect(
      getShownNotices([long], 10_000_000, { firstSeen: arrivalTimes }).map((each) => each.id),
    ).toEqual(['repaired'])
    expect(getFinishedNotices([long], arrivalTimes, 10_000_000)).toEqual([])
  })

  it('names the ones whose caller may dismiss them, and only those', () => {
    const saidText: Notice = { id: 'renamed', text: 'Renamed', stay: 'read' }
    const retainedNotice: Notice = { id: 'occupied', text: 'Filed there already', stay: 'kept' }
    const work: Notice = { id: 'embedding', text: 'Indexing', isWorking: true }
    const all = [saidText, retainedNotice, work]
    const arrivalTimes = getArrivalTimes(new Map(), all, 0)

    expect(getFinishedNotices(all, arrivalTimes, 0)).toEqual([])
    expect(getFinishedNotices(all, arrivalTimes, 1_000_000)).toEqual(['renamed'])
  })
})

describe('how many cards stand at once', () => {
  const work = (id: string): Notice => ({ id, text: 'Indexing', isWorking: true })
  const word = (id: string): Notice => ({ id, text: 'Renamed', stay: 'read' })

  it('folds nothing while there is room', () => {
    expect(foldNotices([work('a'), word('b')], 4)).toEqual({
      visible: [work('a'), word('b')],
      overflowCount: 0,
    })
  })

  it('folds nothing while there is room, even when everything in it could fold', () => {
    const renderedNotices = [word('a'), word('b'), word('c')]

    expect(foldNotices(renderedNotices, 4)).toEqual({ visible: renderedNotices, overflowCount: 0 })
  })

  it('never folds a notice that holds', () => {
    const persistentNotice = (id: string): Notice => ({ id, text: 'Indexing', stay: 'holds' })
    const renderedNotices = [persistentNotice('a'), persistentNotice('b'), persistentNotice('c')]

    expect(foldNotices(renderedNotices, 1)).toEqual({ visible: renderedNotices, overflowCount: 0 })
  })

  it('folds the oldest of what has been returned, and counts them', () => {
    const renderedNotices = [word('a'), word('b'), work('c'), word('d')]

    expect(foldNotices(renderedNotices, 2)).toEqual({
      visible: [work('c'), word('d')],
      overflowCount: 2,
    })
  })

  it('folds no work and nothing that is so, however many there are', () => {
    const renderedNotices = [work('a'), work('b'), work('c'), work('d'), work('e')]

    expect(foldNotices(renderedNotices, 2)).toEqual({ visible: renderedNotices, overflowCount: 0 })
  })

  it('keeps the order of what it did not fold', () => {
    const renderedNotices = [word('a'), word('b'), word('c')]

    expect(foldNotices(renderedNotices, 2).visible.map((each: Notice) => each.id)).toEqual([
      'b',
      'c',
    ])
  })

  it('folds nothing that stopped badly, wherever it remains', () => {
    // Trouble is what a person has to see, and it arrives before the reports
    // that pile up behind it.
    const failed: Notice = { id: 'failed', text: 'Reading', tone: 'alarm', stay: 'kept' }
    const renderedNotices = [failed, word('a'), word('b'), word('c')]

    const { visible, overflowCount } = foldNotices(renderedNotices, 2)

    expect(visible.map((each: Notice) => each.id)).toEqual(['failed', 'c'])
    expect(overflowCount).toBe(2)
  })
})

describe('measuring how fast a count moves', () => {
  const createFetching = (count: number): Notice => ({
    id: 'model',
    text: 'Preparing the model',
    done: count,
    total: 470_268_510,
    counting: 'bytes',
    isWorking: true,
  })

  it('reads a count once before it has a rate', () => {
    const movements = measureMovement(new Map(), [createFetching(0)], 1000)

    expect(movements.get('model')).toEqual({ completedCount: 0, rate: 0, timestamp: 1000 })
  })

  it('measures how fast the count is moving between two samples', () => {
    const first = measureMovement(new Map(), [createFetching(0)], 1000)
    const second = measureMovement(first, [createFetching(4_000_000)], 3000)

    expect(second.get('model')?.rate).toBe(2_000_000)
  })

  it('measures a group over the stretch it took, not the moment it landed in', () => {
    // work did four hundred thousand a second, whatever the last reading saw.
    let movements = measureMovement(new Map(), [createFetching(0)], 1000)
    for (let time = 1250; time < 11_000; time += 250) {
      movements = measureMovement(movements, [createFetching(0)], time)
    }
    movements = measureMovement(movements, [createFetching(4_000_000)], 11_000)

    expect(movements.get('model')?.rate).toBe(400_000)
  })

  it('forgets work that is no longer standing', () => {
    const was: ReadonlyMap<string, Movement> = new Map([
      ['gone', { completedCount: 5, rate: 1, timestamp: 0 }],
    ])

    expect(measureMovement(was, [createFetching(0)], 1000).has('gone')).toBe(false)
  })

  it('measures nothing for work with no total to count against', () => {
    const nothing: Notice = { id: 'scan', text: 'Recognising a scan', isWorking: true }

    expect(measureMovement(new Map(), [nothing], 1000).size).toBe(0)
  })
})
