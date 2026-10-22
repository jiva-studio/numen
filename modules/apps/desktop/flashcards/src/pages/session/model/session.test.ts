import { afterEach, describe, expect, it, vi } from 'vitest'

import { useReviewSession } from './session'
import type { SessionClient, SessionStart } from './session'

/** One card as the application hands it over. */
const createCard = (mark: string) => ({
  deck: 'decks/Words.md',
  section: '',
  mark,
  face: 'Say it',
  heading: mark,
  front: `<p>${mark}</p>`,
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
 * createSession is the application as a test holds it: what it was asked, and
 * what it answers. An answer that never settles is what a slow disk looks like
 * from here, so a second press arrives while the first is still being written.
 */
function createSession(how?: { answering?: Promise<{ answer: string }>; refuses?: unknown }) {
  const asks: { what: string; said: unknown }[] = []
  let content = 0
  const cards: SessionClient = {
    async startSession(one) {
      asks.push({ what: 'start', said: one })
      return opening('one', 'two', 'three')
    },
    async answerCard(one) {
      asks.push({ what: 'answer', said: one })
      if (how?.refuses) throw how.refuses
      if (how?.answering) return how.answering
      content += 1
      return { answer: `01${content}` }
    },
    async takeBackAnswer(one) {
      asks.push({ what: 'takeBack', said: one })
      if (how?.refuses) throw how.refuses
      return {}
    },
  }
  const errors: unknown[] = []
  const one = useReviewSession({ cards, reportError: (why) => errors.push(why), now: () => 1000 })
  return { one, asks, errors }
}

// A session is opened over a deck, over the whole vault, or over one preset.
// The empty path is a preset of its own — the one scheduling the decks that
// name none — so it is told apart from naming no preset at all.
describe('what a session is opened over', () => {
  it('names no preset where it is opened over decks', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', 'decks/Words.md')

    expect(asks[0]?.said).toStrictEqual({ vault: '01VAULT', deck: 'decks/Words.md' })
  })

  it('names the preset where it is opened over one', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', '', 'Sanskrit.md')

    expect(asks[0]?.said).toStrictEqual({ vault: '01VAULT', deck: '', preset: 'Sanskrit.md' })
  })

  it('names the defaults by the empty path, and not by naming nothing', async () => {
    const { one, asks } = createSession()

    await one.start('01VAULT', '', '')

    expect(asks[0]?.said).toStrictEqual({ vault: '01VAULT', deck: '', preset: '' })
  })

  // A preset with nothing to ask fails, and a tile standing for one is not
  // pressed. A count read a moment ago can still be overtaken, and what comes
  // back is said and nothing is opened.
  it('opens nothing where the preset fails, and says why', async () => {
    const cards: SessionClient = {
      startSession: () =>
        Promise.reject(new Error('this preset schedules nothing today: it is paused')),
      answerCard: () => Promise.reject(new Error('no')),
      takeBackAnswer: () => Promise.reject(new Error('no')),
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
    expect(one.asked.value).toHaveLength(3)
    expect(one.card.value?.mark).toBe('one')
    expect(one.shown.value).toBe(false)
    expect(one.left.value).toBe(3)
    expect(one.over.value).toBe(false)
  })

  it('is release whole, so no card of it can be drawn again', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    one.forget()

    expect(one.card.value).toBeNull()
    expect(one.asked.value).toHaveLength(0)
    expect(one.run.value).toBe('')
    expect(one.shown.value).toBe(false)
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

    expect(one.at.value).toBe(1)
    expect(one.answers.value).toEqual(['011'])
  })

  it('leaves the card where it was when the answer could not be written', async () => {
    const { one, asks, errors } = createSession({ refuses: new Error('the disk is full') })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    expect(errors).toHaveLength(1)
    expect(one.at.value).toBe(0)
    expect(one.card.value?.mark).toBe('one')
    expect(one.shown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
  })

  it('is not answered before the card is turned over', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    await one.answer('good')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(0)
    expect(one.at.value).toBe(0)
  })

  it('brings the card back with its answer showing when one is taken back', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')
    expect(one.card.value?.mark).toBe('two')

    await one.takeBack()

    expect(one.card.value?.mark).toBe('one')
    expect(one.shown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(one.done.value).toBe(0)
  })

  it('takes nothing back before anything was answered', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    await one.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack')).toHaveLength(0)
    expect(one.at.value).toBe(0)
  })

  it('keeps the answer it wrote when taking it back failed', async () => {
    let refuses = false
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async answerCard() {
        return { answer: '011' }
      },
      async takeBackAnswer() {
        if (refuses) throw new Error('gone')
        return {}
      },
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')
    expect(one.answers.value).toHaveLength(1)
    expect(one.at.value).toBe(1)

    refuses = true
    await one.takeBack()

    // The vault holds the answer; the window may not pretend otherwise.
    expect(errors).toHaveLength(1)
    expect(one.answers.value).toHaveLength(1)
    expect(one.at.value).toBe(1)
    expect(one.shown.value).toBe(false)
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
    expect(one.done.value).toBe(3)
  })

  it('says nothing was opened when the vault could not be sat down to', async () => {
    const cards: SessionClient = {
      startSession: () => Promise.reject(new Error('this folder cannot be read as a vault')),
      answerCard: () => Promise.reject(new Error('no')),
      takeBackAnswer: () => Promise.reject(new Error('no')),
    }
    const errors: unknown[] = []
    const one = useReviewSession({ cards, reportError: (why) => errors.push(why) })

    expect(await one.start('01VAULT', '')).toBeNull()
    expect(errors).toHaveLength(1)
    expect(one.card.value).toBeNull()
  })

  it('measures how long the card stood in front of the person', async () => {
    let clock = 1000
    const asks: { said: unknown }[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async answerCard(one) {
        asks.push({ said: one })
        return { answer: '011' }
      },
      async takeBackAnswer() {
        return {}
      },
    }
    const one = useReviewSession({ cards, reportError: () => {}, now: () => clock })

    clock += 1000
    await one.start('01VAULT', '')
    clock += 4200
    one.show()
    await one.answer('good')

    expect(asks[0]?.said).toMatchObject({ tookMs: 4200n })

    clock += 3000
    one.show()
    await one.answer('good')

    expect(asks[1]?.said).toMatchObject({ tookMs: 3000n })

    await one.takeBack()
    clock += 1500
    await one.answer('good')

    expect(asks[2]?.said).toMatchObject({ tookMs: 1500n })
  })
})

describe('the next card', () => {
  it('arrives up with its answer hidden', async () => {
    const { one } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    expect(one.card.value?.mark).toBe('two')
    expect(one.shown.value).toBe(false)
  })
})

describe('an answer taken back', () => {
  it('notifies the vault which answer of which run it was', async () => {
    const { one, asks } = createSession()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    await one.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack').map((ask) => ask.said)).toEqual([
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

    expect(one.card.value?.mark).toBe('two')
    expect(one.answers.value).toEqual(['011'])
  })

  // The answer is taken out of the vault once, whatever a second press says.
  it('is taken back once, whatever a second press says', async () => {
    let settle = () => {}
    const asks: string[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two', 'three')
      },
      async answerCard() {
        return { answer: `01${asks.length}` }
      },
      takeBackAnswer(one) {
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
    expect(one.card.value?.mark).toBe('one')
  })

  it('leaves the session free to answer again once it fails', async () => {
    let refuses = true
    const cards: SessionClient = {
      async startSession() {
        return opening('one', 'two')
      },
      async answerCard() {
        return { answer: '011' }
      },
      async takeBackAnswer() {
        if (refuses) throw new Error('gone')
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

    refuses = false
    await one.takeBack()
    expect(one.card.value?.mark).toBe('one')
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
              mark: '3f4g5h6j7k',
              face: 'Say it',
              heading: 'Leaf mould',
              front: '<p>Leaf mould</p>',
              back: '<p>Compost made of fallen leaves alone</p>',
              isNew: false,
              intervals: { again: 60n, hard: 600n, good: 86400n, easy: 345600n },
            },
          ],
          unwritten: [],
          skipped: 0,
        }
      },
      answerCard: () => Promise.reject(new Error('no')),
      takeBackAnswer: () => Promise.reject(new Error('no')),
    }
    const one = useReviewSession({ cards, reportError: () => {} })

    await one.start('01VAULT', '')

    expect(one.asked.value).toEqual([
      {
        deck: 'decks/Words.md',
        section: 'Garden',
        mark: '3f4g5h6j7k',
        face: 'Say it',
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
    const response: { tookMs: bigint }[] = []
    const cards: SessionClient = {
      async startSession() {
        return opening('one')
      },
      async answerCard(one) {
        response.push(one)
        return { answer: '011' }
      },
      async takeBackAnswer() {
        return {}
      },
    }
    const one = useReviewSession({ cards, reportError: () => {} })

    await one.start('01VAULT', '')
    vi.setSystemTime(new Date('2026-03-01T09:00:03Z'))
    one.show()
    await one.answer('good')

    expect(response.map((one) => one.tookMs)).toEqual([3000n])
  })
})
