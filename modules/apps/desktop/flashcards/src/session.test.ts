import { describe, expect, it } from 'vitest'

import { session } from './session'
import type { Asking, Opened } from './session'

/** One card as the application hands it over. */
const asked = (card: string) => ({
  deck: 'decks/Words.md',
  section: '',
  card,
  face: 'Say it',
  heading: card,
  front: `<p>${card}</p>`,
  back: '<p>and back</p>',
  seen: false,
  ahead: undefined,
})

/** opening is what starting a sitting comes back with. */
const opening = (...cards: string[]): Opened => ({
  run: 'flashcards/01A.jsonl',
  asked: cards.map(asked),
  unwritten: [],
  skipped: 0,
})

/**
 * held is the application as a test holds it: what it was asked, and what it
 * answers. An answer that never settles is what a slow disk looks like from
 * here, so a second press arrives while the first is still being written.
 */
function held(said?: { answering?: Promise<{ answer: string }>; refuses?: unknown }) {
  const asks: { what: string; said: unknown }[] = []
  let written = 0
  const cards: Asking = {
    async start(one) {
      asks.push({ what: 'start', said: one })
      return opening('one', 'two', 'three')
    },
    async answer(one) {
      asks.push({ what: 'answer', said: one })
      if (said?.refuses) throw said.refuses
      if (said?.answering) return said.answering
      written += 1
      return { answer: `01${written}` }
    },
    async takeBack(one) {
      asks.push({ what: 'takeBack', said: one })
      if (said?.refuses) throw said.refuses
      return {}
    },
  }
  const trouble: unknown[] = []
  const one = session({ cards, failed: (why) => trouble.push(why), now: () => 1000 })
  return { one, asks, trouble }
}

// A sitting is opened over a deck, over the whole vault, or over one preset.
// The empty path is a preset of its own — the one scheduling the decks that
// name none — so it is told apart from naming no preset at all.
describe('what a sitting is opened over', () => {
  it('names no preset where it is opened over decks', async () => {
    const { one, asks } = held()

    await one.start('01VAULT', 'decks/Words.md')

    expect(asks[0]?.said).toStrictEqual({ vaultId: '01VAULT', deck: 'decks/Words.md' })
  })

  it('names the preset where it is opened over one', async () => {
    const { one, asks } = held()

    await one.start('01VAULT', '', 'Sanskrit.md')

    expect(asks[0]?.said).toStrictEqual({ vaultId: '01VAULT', deck: '', preset: 'Sanskrit.md' })
  })

  it('names the defaults by the empty path, and not by naming nothing', async () => {
    const { one, asks } = held()

    await one.start('01VAULT', '', '')

    expect(asks[0]?.said).toStrictEqual({ vaultId: '01VAULT', deck: '', preset: '' })
  })

  // A preset with nothing to ask is refused, and a tile standing for one is not
  // pressed. A count read a moment ago can still be overtaken, and what comes
  // back is said and nothing is opened.
  it('opens nothing where the preset is refused, and says why', async () => {
    const cards: Asking = {
      start: () => Promise.reject(new Error('this preset schedules nothing today: it is paused')),
      answer: () => Promise.reject(new Error('no')),
      takeBack: () => Promise.reject(new Error('no')),
    }
    const trouble: unknown[] = []
    const one = session({ cards, failed: (why) => trouble.push(why) })

    expect(await one.start('01VAULT', '', 'Sanskrit.md')).toBeNull()
    expect(String(trouble[0])).toContain('this preset schedules nothing today')
    expect(one.run.value).toBe('')
    expect(one.card.value).toBeNull()
  })
})

describe('a sitting', () => {
  it('opens on the cards it was handed, at the first of them', async () => {
    const { one } = held()
    const report = await one.start('01VAULT', '')

    expect(report).toEqual({ unwritten: [], skipped: 0 })
    expect(one.asked.value).toHaveLength(3)
    expect(one.card.value?.card).toBe('one')
    expect(one.shown.value).toBe(false)
    expect(one.left.value).toBe(3)
    expect(one.over.value).toBe(false)
  })

  it('is let go of whole, so no card of it can be drawn again', async () => {
    const { one } = held()
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
    const { one, asks } = held({ answering })

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
    const { one, asks, trouble } = held({ refuses: new Error('the disk is full') })
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    expect(trouble).toHaveLength(1)
    expect(one.at.value).toBe(0)
    expect(one.card.value?.card).toBe('one')
    expect(one.shown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(1)
  })

  it('is not answered before the card is turned over', async () => {
    const { one, asks } = held()
    await one.start('01VAULT', '')
    await one.answer('good')

    expect(asks.filter((ask) => ask.what === 'answer')).toHaveLength(0)
    expect(one.at.value).toBe(0)
  })

  it('brings the card back with its answer showing when one is taken back', async () => {
    const { one } = held()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')
    expect(one.card.value?.card).toBe('two')

    await one.takeBack()

    expect(one.card.value?.card).toBe('one')
    expect(one.shown.value).toBe(true)
    expect(one.answers.value).toEqual([])
    expect(one.done.value).toBe(0)
  })

  it('takes nothing back before anything was answered', async () => {
    const { one, asks } = held()
    await one.start('01VAULT', '')
    await one.takeBack()

    expect(asks.filter((ask) => ask.what === 'takeBack')).toHaveLength(0)
    expect(one.at.value).toBe(0)
  })

  it('keeps the answer it wrote when taking it back was refused', async () => {
    const { one } = held()
    await one.start('01VAULT', '')
    one.show()
    await one.answer('good')

    const { one: other } = held({ refuses: new Error('gone') })
    await other.start('01VAULT', '')
    other.show()
    await other.answer('good')

    // The vault holds the answer; the window may not pretend otherwise.
    expect(other.answers.value).toHaveLength(0)
    expect(one.answers.value).toHaveLength(1)
  })

  it('is over when the last card has been answered', async () => {
    const { one } = held()
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
    const cards: Asking = {
      start: () => Promise.reject(new Error('this vault has not been read yet')),
      answer: () => Promise.reject(new Error('no')),
      takeBack: () => Promise.reject(new Error('no')),
    }
    const trouble: unknown[] = []
    const one = session({ cards, failed: (why) => trouble.push(why) })

    expect(await one.start('01VAULT', '')).toBeNull()
    expect(trouble).toHaveLength(1)
    expect(one.card.value).toBeNull()
  })

  it('measures how long the card stood in front of the person', async () => {
    let clock = 1000
    const asks: { said: unknown }[] = []
    const cards: Asking = {
      async start() {
        return opening('one')
      },
      async answer(one) {
        asks.push({ said: one })
        return { answer: '011' }
      },
      async takeBack() {
        return {}
      },
    }
    const one = session({ cards, failed: () => {}, now: () => clock })

    await one.start('01VAULT', '')
    clock += 4200
    one.show()
    await one.answer('good')

    expect(asks[0]?.said).toMatchObject({ tookMs: 4200n })
  })
})
