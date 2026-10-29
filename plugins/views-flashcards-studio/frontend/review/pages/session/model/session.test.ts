import { afterEach, describe, expect, it, vi } from 'vitest'

import { useReviewSession } from './session'
import type { SessionClient, SessionStart } from './session'

/** One card as the application returns it. */
const createCard = (cardId: string) => ({
  deck: 'decks/Words.md',
  section: '',
  cardId,
  template: 'Say it',
  heading: cardId,
  front: `<p>${cardId}</p>`,
  back: '<p>and back</p>',
  isNew: true,
  intervals: undefined,
})

/** opening is what starting a session comes back with. */
const opening = (...cards: string[]): SessionStart => ({
  run: 'flashcards/01A.jsonl',
  cards: cards.map(createCard),
  unwritten: [],
  skipped: 0,
})

/**
 * createSession is the application as a test models it: the requests it
 * received and the answers it returns. An answer that never resolves simulates
 * a slow disk, so a second press arrives while the first is still being written.
 */
function createSession(how?: { answering?: Promise<{ answer: string }>; failure?: unknown }) {
  const asks: { what: string; request: unknown }[] = []
  let content = 0
  const cards: SessionClient = {
    async startSession(one) {
      asks.push({ what: 'start', request: one })
      return opening('one', 'two', 'three')
    },
    async recordReview(one) {
      asks.push({ what: 'answer', request: one })
      if (how?.failure) throw how.failure
      if (how?.answering) return how.answering
      content += 1
      return { answer: `01${content}` }
    },
    async undoReview(one) {
      asks.push({ what: 'takeBack', request: one })
      if (how?.failure) throw how.failure
      return {}
    },
  }
  const errors: unknown[] = []
  const one = useReviewSession({ cards, reportError: (why) => errors.push(why), now: () => 1000 })
  return { one, asks, errors }
}

// A session is opened over a deck, over the whole vault, or over one preset.
// The empty path is a preset of its own — the one scheduling the decks that
// name none — so it is distinct from naming no preset at all.
describe('what a session is opened over', () => {
  it('names no preset where it is opened over decks', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', 'decks/Words.md')

    expect(asks[0]?.request).toStrictEqual({ vault: '01VAULT', deck: 'decks/Words.md' })
  })

  it('names the preset where it is opened over one', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', '', 'Sanskrit.md')

    expect(asks[0]?.request).toStrictEqual({ vault: '01VAULT', deck: '', preset: 'Sanskrit.md' })
  })

  it('names the defaults by the empty path, and not by naming nothing', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', '', '')

    expect(asks[0]?.request).toStrictEqual({ vault: '01VAULT', deck: '', preset: '' })
  })

  // A preset with nothing to ask fails, and a tile standing for one is not
  // pressed. A count read a moment ago can still be overtaken, and what comes
  // back is reported and nothing is opened.
  it('opens nothing where the preset fails, and says why', async () => {
    const cards: SessionClient = {
      startSession: () =>
        Promise.reject(new Error('this preset schedules nothing today: it is paused')),
      recordReview: () => Promise.reject(new Error('no')),
      undoReview: () => Promise.reject(new Error('no')),
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })

    expect(await one.start('01VAULT', '', 'Sanskrit.md')).toBeNull()
    expect(String(errors[0])).toContain('this preset schedules nothing today')
    expect(one.run.value).toBe('')
    expect(one.card.value).toBeNull()
  })
})

describe('a session', () => {
  it('opens on the cards it was handed, at the first of them', async () => {
    const { one } = createSession()
    const report = await one.start('01VAULT', '')

    expect(report).toEqual({ unwritten: [], skipped: 0 })
    expect(one.cardList.value).toHaveLength(3)
    expect(one.card.value?.cardId).toBe('one')
    expect(one.isAnswerShown.value).toBe(false)
    expect(one.left.value).toBe(3)
    expect(one.over.value).toBe(false)
  })

  it('is release whole, so no card of it can be drawn again', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    one.forget()

    expect(one.card.value).toBeNull()
    expect(one.cardList.value).toHaveLength(0)
    expect(one.run.value).toBe('')
    expect(one.isAnswerShown.value).toBe(false)
  })

  it('answers a card once, whatever a second press says', async () => {
    // The first answer is still being written when the second is pressed.
    let settle = (_: { answer: string }) => {}
    const answering = new Promise<{ answer: string }>((then) => {
      settle = then
    })
    const { one, asks } = createSession({ answering })

    await one.start('01VAULT', '')
    one.show()
    const first = one.answer('good')
    await one.answer('again')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
    settle({ answer: '011' })
    await first

    expect(one.cardIndex.value).toBe(1)
    expect(one.answers.value).toEqual(['011'])
  })

  it('leaves the card where it was when the answer could not be written', async () => {
    const { one, asks, errors } = createSession({ failure: new Error('the disk is full') })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    expect(errors).toHaveLength(1)
    expect(one.cardIndex.value).toBe(0)
    expect(one.card.value?.cardId).toBe('one')
    expect(one.isAnswerShown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
  })

  it('is not answered before the card is turned over', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    await one.answer('good')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(0)
    expect(one.cardIndex.value).toBe(0)
  })

  it('brings the card back with its answer showing when one is taken back', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')
    expect(one.card.value?.cardId).toBe('two')

    await one.takeBack()

    expect(one.card.value?.cardId).toBe('one')
    expect(one.isAnswerShown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(one.completedCount.value).toBe(0)
  })

  it('takes nothing back before anything was answered', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    await one.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack')).toHaveLength(0)
    expect(one.cardIndex.value).toBe(0)
  })

  it('keeps the answer it wrote when taking it back failed', async () => {
    let failure = false
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async recordReview() {
        return { answer: '011' }
      },
      async undoReview() {
        if (failure) throw new Error('gone')
        return {}
      },
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')
    expect(one.answers.value).toHaveLength(1)
    expect(one.cardIndex.value).toBe(1)

    failure = true
    await one.takeBack()

    // The vault holds the answer, so the window keeps it as recorded.
    expect(errors).toHaveLength(1)
    expect(one.answers.value).toHaveLength(1)
    expect(one.cardIndex.value).toBe(1)
    expect(one.isAnswerShown.value).toBe(false)
  })

  it('is over when the last card has been answered', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    for (const _ of [0, 1, 2]) {
      one.show()
      await one.answer('good')
    }

    expect(one.over.value).toBe(true)
    expect(one.left.value).toBe(0)
    expect(one.completedCount.value).toBe(3)
  })

  it('says nothing was opened when the vault could not be sat down to', async () => {
    const cards: SessionClient = {
      startSession: () => Promise.reject(new Error('this folder cannot be read as a vault')),
      recordReview: () => Promise.reject(new Error('no')),
      undoReview: () => Promise.reject(new Error('no')),
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })

    expect(await one.start('01VAULT', '')).toBeNull()
    expect(errors).toHaveLength(1)
    expect(one.card.value).toBeNull()
  })

  it('measures how long the card stood in front of the person', async () => {
    let clock = 1000
    const asks: { request: unknown }[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async recordReview(one) {
        asks.push({ request: one })
        return { answer: '011' }
      },
      async undoReview() {
        return {}
      },
    }
    const one = useReviewSession({ cards, reportError: () => {}, now: () => clock })

    clock += 1000
    await one.start('01VAULT', '')
    clock += 4200
    one.show()
    await one.answer('good')

    expect(asks[0]?.request).toMatchObject({ durationMs: 4200 })

    clock += 3000
    one.show()
    await one.answer('good')

    expect(asks[1]?.request).toMatchObject({ durationMs: 3000 })

    await one.takeBack()
    clock += 1500
    await one.answer('good')

    expect(asks[2]?.request).toMatchObject({ durationMs: 1500 })
  })
})

describe('the next card', () => {
  it('arrives up with its answer hidden', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    expect(one.card.value?.cardId).toBe('two')
    expect(one.isAnswerShown.value).toBe(false)
  })
})

describe('an answer taken back', () => {
  it('notifies the vault which answer of which run it was', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    await one.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack').map((ask) => ask.request)).toEqual([
      { vault: '01VAULT', run: 'flashcards/01A.jsonl', answer: '011' },
    ])
  })

  it('brings back the card before the one in front of the person', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    for (const _ of [0, 1]) {
      one.show()
      await one.answer('good')
    }

    await one.takeBack()

    expect(one.card.value?.cardId).toBe('two')
    expect(one.answers.value).toEqual(['011'])
  })

  // The answer is withdrawn from the vault once, regardless of a second press.
  it('is taken back once, whatever a second press says', async () => {
    let settle = () => {}
    const asks: string[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two', 'three')
      },
      async recordReview() {
        return { answer: `01${asks.length}` }
      },
      undoReview(one) {
        asks.push(one.answer)
        return new Promise<void>((then) => {
          settle = then
        })
      },
    }
    const one = useReviewSession({ cards, reportError: () => {}, now: () => 1000 })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    const first = one.takeBack()
    expect(one.writing.value).toBe(true)
    await one.takeBack()
    expect(asks).toHaveLength(1)

    settle()
    await first
    expect(one.writing.value).toBe(false)
    expect(one.card.value?.cardId).toBe('one')
  })

  it('leaves the session free to answer again once it fails', async () => {
    let failure = true
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async recordReview() {
        return { answer: '011' }
      },
      async undoReview() {
        if (failure) throw new Error('gone')
        return {}
      },
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why), now: () => 1000 })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    await one.takeBack()
    expect(errors).toHaveLength(1)
    expect(one.writing.value).toBe(false)

    failure = false
    await one.takeBack()
    expect(one.card.value?.cardId).toBe('one')
  })
})

describe('the cards a session opens on', () => {
  it('holds each as it was handed over, with the intervals ahead of it in seconds', async () => {
    const cards: SessionClient = {
      async startSession() {
        return {
          run: 'flashcards/01A.jsonl',
          cards: [
            {
              deck: 'decks/Words.md',
              section: 'Garden',
              cardId: '3f4g5h6j7k',
              template: 'Say it',
              heading: 'Leaf mould',
              front: '<p>Leaf mould</p>',
              back: '<p>Compost made of fallen leaves alone</p>',
              isNew: false,
              intervals: { again: 60, hard: 600, good: 86400, easy: 345600 },
            },
          ],
          unwritten: [],
          skipped: 0,
        }
      },
      recordReview: () => Promise.reject(new Error('no')),
      undoReview: () => Promise.reject(new Error('no')),
    }
    const one = useReviewSession({ cards, reportError: () => {} })

    await one.start('01VAULT', '')

    expect(one.cardList.value).toEqual([
      {
        deck: 'decks/Words.md',
        section: 'Garden',
        cardId: '3f4g5h6j7k',
        template: 'Say it',
        heading: 'Leaf mould',
        front: '<p>Leaf mould</p>',
        back: '<p>Compost made of fallen leaves alone</p>',
        isNew: false,
        ahead: { again: 60, hard: 600, good: 86400, easy: 345600 },
      },
    ])
  })
})

describe('a session keeping its own time', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('measures card review duration using timer', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-01T09:00:00Z'))
    const response: { durationMs: number }[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one')
      },
      async recordReview(one) {
        response.push(one)
        return { answer: '011' }
      },
      async undoReview() {
        return {}
      },
    }
    const one = useReviewSession({ cards, reportError: () => {} })

    await one.start('01VAULT', '')
    vi.setSystemTime(new Date('2026-03-01T09:00:03Z'))
    one.show()
    await one.answer('good')

    expect(response.map((one) => one.durationMs)).toEqual([3000])
  })
})
