import { afterEach, describe, expect, it, vi } from 'vitest'

import { useReviewSession } from './session'
import type { SessionClient, SessionStart } from './session'
import type { UndoReviewResponse } from '../../../lib/wire/review'

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

const opening = (...cards: string[]): SessionStart => ({
  run: 'flashcards/01A.jsonl',
  cards: cards.map(createCard),
  unwritten: [],
  skipped: 0,
})

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

    await one.opener.start('01VAULT', 'decks/Words.md')

    expect(asks[0]?.request).toStrictEqual({ vault: '01VAULT', deck: 'decks/Words.md' })
  })

  it('names the preset where it is opened over one', async () => {
    const { one, asks } = createSession()

    await one.opener.start('01VAULT', '', 'Sanskrit.md')

    expect(asks[0]?.request).toStrictEqual({ vault: '01VAULT', deck: '', preset: 'Sanskrit.md' })
  })

  it('names the defaults by the empty path, and not by naming nothing', async () => {
    const { one, asks } = createSession()

    await one.opener.start('01VAULT', '', '')

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

    expect(await one.opener.start('01VAULT', '', 'Sanskrit.md')).toBeNull()
    expect(String(errors[0])).toContain('this preset schedules nothing today')
    expect(one.ids.run.value).toBe('')
    expect(one.cards.queue.card.value).toBeNull()
  })
})

describe('a session', () => {
  it('opens on the cards it was handed, at the first of them', async () => {
    const { one } = createSession()
    const report = await one.opener.start('01VAULT', '')

    expect(report).toEqual({ unwritten: [], skipped: 0 })
    expect(one.cards.queue.cardList.value).toHaveLength(3)
    expect(one.cards.queue.card.value?.address.cardId).toBe('one')
    expect(one.cards.queue.isAnswerShown.value).toBe(false)
    expect(one.cards.progress.left.value).toBe(3)
    expect(one.cards.progress.isOver.value).toBe(false)
  })

  it('is release whole, so no card of it can be drawn again', async () => {
    const { one } = createSession()
    await one.opener.start('01VAULT', '')
    one.show()
    one.opener.forget()

    expect(one.cards.queue.card.value).toBeNull()
    expect(one.cards.queue.cardList.value).toHaveLength(0)
    expect(one.ids.run.value).toBe('')
    expect(one.cards.queue.isAnswerShown.value).toBe(false)
  })

  it('answers a card once, whatever a second press says', async () => {
    // The first answer is still being written when the second is pressed.
    let settle: (_: { answer: string }) => void = (_: { answer: string }) => undefined
    const answering = new Promise<{ answer: string }>((then) => {
      settle = then
    })
    const { one, asks } = createSession({ answering })

    await one.opener.start('01VAULT', '')
    one.show()
    const first = one.answering.answer('good')
    await one.answering.answer('again')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
    settle({ answer: '011' })
    await first

    expect(one.cards.queue.cardIndex.value).toBe(1)
    expect(one.cards.queue.answers.value).toEqual(['011'])
  })

  it('leaves the card where it was when the answer could not be written', async () => {
    const { one, asks, errors } = createSession({ failure: new Error('the disk is full') })
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')

    expect(errors).toHaveLength(1)
    expect(one.cards.queue.cardIndex.value).toBe(0)
    expect(one.cards.queue.card.value?.address.cardId).toBe('one')
    expect(one.cards.queue.isAnswerShown.value).toBe(true)
    expect(one.cards.queue.answers.value).toEqual([])
    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
  })

  it('is not answered before the card is turned over', async () => {
    const { one, asks } = createSession()
    await one.opener.start('01VAULT', '')
    await one.answering.answer('good')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(0)
    expect(one.cards.queue.cardIndex.value).toBe(0)
  })

  it('brings the card back with its answer showing when one is taken back', async () => {
    const { one } = createSession()
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')
    expect(one.cards.queue.card.value?.address.cardId).toBe('two')

    await one.answering.takeBack()

    expect(one.cards.queue.card.value?.address.cardId).toBe('one')
    expect(one.cards.queue.isAnswerShown.value).toBe(true)
    expect(one.cards.queue.answers.value).toEqual([])
    expect(one.cards.progress.completedCount.value).toBe(0)
  })

  it('takes nothing back before anything was answered', async () => {
    const { one, asks } = createSession()
    await one.opener.start('01VAULT', '')
    await one.answering.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack')).toHaveLength(0)
    expect(one.cards.queue.cardIndex.value).toBe(0)
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
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')
    expect(one.cards.queue.answers.value).toHaveLength(1)
    expect(one.cards.queue.cardIndex.value).toBe(1)

    failure = true
    await one.answering.takeBack()

    // The vault holds the answer, so the window keeps it as recorded.
    expect(errors).toHaveLength(1)
    expect(one.cards.queue.answers.value).toHaveLength(1)
    expect(one.cards.queue.cardIndex.value).toBe(1)
    expect(one.cards.queue.isAnswerShown.value).toBe(false)
  })

  it('is over when the last card has been answered', async () => {
    const { one } = createSession()
    await one.opener.start('01VAULT', '')
    for (const _ of [0, 1, 2]) {
      one.show()
      await one.answering.answer('good')
    }

    expect(one.cards.progress.isOver.value).toBe(true)
    expect(one.cards.progress.left.value).toBe(0)
    expect(one.cards.progress.completedCount.value).toBe(3)
  })

  it('says nothing was opened when the vault could not be sat down to', async () => {
    const cards: SessionClient = {
      startSession: () => Promise.reject(new Error('this folder cannot be read as a vault')),
      recordReview: () => Promise.reject(new Error('no')),
      undoReview: () => Promise.reject(new Error('no')),
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })

    expect(await one.opener.start('01VAULT', '')).toBeNull()
    expect(errors).toHaveLength(1)
    expect(one.cards.queue.card.value).toBeNull()
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
    const one = useReviewSession({ cards, reportError: () => undefined, now: () => clock })

    clock += 1000
    await one.opener.start('01VAULT', '')
    clock += 4200
    one.show()
    await one.answering.answer('good')

    expect(asks[0]?.request).toMatchObject({ durationMs: 4200 })

    clock += 3000
    one.show()
    await one.answering.answer('good')

    expect(asks[1]?.request).toMatchObject({ durationMs: 3000 })

    await one.answering.takeBack()
    clock += 1500
    await one.answering.answer('good')

    expect(asks[2]?.request).toMatchObject({ durationMs: 1500 })
  })
})

describe('the next card', () => {
  it('arrives up with its answer hidden', async () => {
    const { one } = createSession()
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')

    expect(one.cards.queue.card.value?.address.cardId).toBe('two')
    expect(one.cards.queue.isAnswerShown.value).toBe(false)
  })
})

describe('an answer taken back', () => {
  it('notifies the vault which answer of which run it was', async () => {
    const { one, asks } = createSession()
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')

    await one.answering.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack').map((ask) => ask.request)).toEqual([
      { vault: '01VAULT', run: 'flashcards/01A.jsonl', answer: '011' },
    ])
  })

  it('brings back the card before the one in front of the person', async () => {
    const { one } = createSession()
    await one.opener.start('01VAULT', '')
    for (const _ of [0, 1]) {
      one.show()
      await one.answering.answer('good')
    }

    await one.answering.takeBack()

    expect(one.cards.queue.card.value?.address.cardId).toBe('two')
    expect(one.cards.queue.answers.value).toEqual(['011'])
  })

  // The answer is withdrawn from the vault once, regardless of a second press.
  it('is taken back once, whatever a second press says', async () => {
    let settle: () => void = () => undefined
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
        return new Promise<UndoReviewResponse>((then) => {
          settle = () => then({})
        })
      },
    }
    const one = useReviewSession({ cards, reportError: () => undefined, now: () => 1000 })
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')

    const first = one.answering.takeBack()
    expect(one.writing.value).toBe(true)
    await one.answering.takeBack()
    expect(asks).toHaveLength(1)

    settle()
    await first
    expect(one.writing.value).toBe(false)
    expect(one.cards.queue.card.value?.address.cardId).toBe('one')
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
    await one.opener.start('01VAULT', '')
    one.show()
    await one.answering.answer('good')

    await one.answering.takeBack()
    expect(errors).toHaveLength(1)
    expect(one.writing.value).toBe(false)

    failure = false
    await one.answering.takeBack()
    expect(one.cards.queue.card.value?.address.cardId).toBe('one')
  })
})

describe('the cards a session opens on', () => {
  it('holds each as it was passed, with the intervals ahead of it in seconds', async () => {
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
    const one = useReviewSession({ cards, reportError: () => undefined })

    await one.opener.start('01VAULT', '')

    expect(one.cards.queue.cardList.value).toEqual([
      {
        address: {
          deck: 'decks/Words.md',
          section: 'Garden',
          cardId: '3f4g5h6j7k',
          template: 'Say it',
        },
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
    const one = useReviewSession({ cards, reportError: () => undefined })

    await one.opener.start('01VAULT', '')
    vi.setSystemTime(new Date('2026-03-01T09:00:03Z'))
    one.show()
    await one.answering.answer('good')

    expect(response.map((one) => one.durationMs)).toEqual([3000])
  })
})
