import { describe, expect, it } from 'vitest'
import { Stopped } from '@numen/protocol'

import { counting } from './counting'
import type { Counted, Counts, Vaulted } from './counting'

const vault = (id: string, said: Partial<Vaulted> = {}): Vaulted => ({
  vaultId: id,
  name: id,
  path: `/vaults/${id}`,
  faces: 3,
  due: 1,
  new: 2,
  decks: [{ deck: 'decks/Words.md', faces: 3, due: 1, new: 2, learned: 1, unbegun: 2 }],
  presets: [
    {
      preset: 'Sanskrit.md',
      title: 'Sanskrit',
      decks: 1,
      cards: 3,
      owedDue: 2,
      owedNew: 1,
      answered: 4,
      answeredNew: 1,
      answeredReviews: 3,
      tookMs: 90000n,
      new: 8,
      reviews: 45,
      minutes: 20,
      closesNew: '',
      closesReviews: '',
      closesMinutes: 'minutes_a_day',
      stopsOn: Stopped.NOTHING,
    },
  ],
  unread: '',
  ...said,
})

/** The vaults as they stand before any of them is counted. */
const listing = (...all: readonly Vaulted[]): Counted => ({
  day: '2026-09-05',
  vaults: all.map((one) => ({ ...vault(one.vaultId), ...one, faces: 0, due: 0, new: 0, decks: [] })),
})

/** One vault's count, as it arrives on its own. */
const count = (one: Vaulted): Counted => ({ day: '', vaults: [], counted: one })

/** A count a test feeds by hand, message by message. */
const feeding = () => {
  const held: Counted[] = []
  let wake: (() => void) | null = null
  let over = false

  const owing = async function* (
    _said: Record<string, never>,
    how?: { signal?: AbortSignal },
  ): AsyncGenerator<Counted> {
    over = false
    for (;;) {
      while (held.length) yield held.shift() as Counted
      if (over) return
      await new Promise<void>((then, stopped) => {
        wake = () => {
          wake = null
          then()
        }
        how?.signal?.addEventListener('abort', () => stopped(new Error('the count was stopped')), {
          once: true,
        })
      })
    }
  }

  return {
    cards: { owing } satisfies Counts,
    says(one: Counted) {
      held.push(one)
      wake?.()
    },
    ends() {
      over = true
      wake?.()
    },
  }
}

/** What is on its way settles. */
const settles = () => new Promise((then) => setTimeout(then, 0))

describe('counting what every vault owes', () => {
  it('holds the vaults before any of them is counted', async () => {
    const front = feeding()
    const one = counting({ cards: front.cards, failed: () => {} })

    void one.count()
    front.says(listing(vault('01A'), vault('01B')))
    await settles()

    expect(one.day.value).toBe('2026-09-05')
    expect(one.vaults.value.map((held) => held.vaultId)).toStrictEqual(['01A', '01B'])
    // Nothing is known about what any of them holds, and none of them reads as
    // a vault owing nothing.
    for (const held of one.vaults.value) {
      expect(held.counted).toBe(false)
      expect(held.due + held.new).toBe(0)
    }
  })

  it('fills each count into its own vault as it lands', async () => {
    const front = feeding()
    const one = counting({ cards: front.cards, failed: () => {} })

    void one.count()
    front.says(listing(vault('01A'), vault('01B')))
    await settles()
    front.says(count(vault('01B', { due: 4, new: 1 })))
    await settles()

    expect(one.vaults.value[0]).toMatchObject({ vaultId: '01A', counted: false, due: 0, new: 0 })
    expect(one.vaults.value[1]).toMatchObject({ vaultId: '01B', counted: true, due: 4, new: 1 })

    front.says(count(vault('01A')))
    await settles()
    expect(one.vaults.value[0]).toMatchObject({ vaultId: '01A', counted: true, due: 1, new: 2 })
    // What a day took arrives in milliseconds and is held in minutes.
    expect(one.vaults.value[0]?.presets[0]).toEqual({
      preset: 'Sanskrit.md',
      title: 'Sanskrit',
      decks: 1,
      cards: 3,
      // What the day leaves is the two the count sends, added up once here.
      owed: 3,
      answered: 4,
      answeredNew: 1,
      answeredReviews: 3,
      took: 1.5,
      new: 8,
      reviews: 45,
      minutes: 20,
      closes: { new: '', reviews: '', minutes: 'minutes_a_day' },
      stopsOn: Stopped.NOTHING,
    })
  })

  it('carries what a vault that could not be counted says, and counts the rest', async () => {
    const front = feeding()
    const one = counting({ cards: front.cards, failed: () => {} })

    void one.count()
    front.says(listing(vault('01A'), vault('01B')))
    front.says(count(vault('01A', { unread: 'this vault has not been read yet' })))
    front.says(count(vault('01B', { due: 6, new: 0 })))
    front.ends()
    await settles()

    expect(one.vaults.value[0]?.unread).toBe('this vault has not been read yet')
    expect(one.vaults.value[1]).toMatchObject({ counted: true, due: 6 })
    expect(one.counting.value).toBe(false)
  })

  it('is still counting until the last of them has arrived', async () => {
    const front = feeding()
    const one = counting({ cards: front.cards, failed: () => {} })

    void one.count()
    front.says(listing(vault('01A')))
    await settles()
    expect(one.counting.value).toBe(true)

    front.says(count(vault('01A')))
    front.ends()
    await settles()
    expect(one.counting.value).toBe(false)
  })

  // The window opening, a sitting ending and a vault moving underneath it all
  // ask, and they arrive together. One count answers all three.
  it('runs one count however many ask for it at once', async () => {
    let asked = 0
    const front = feeding()
    const cards: Counts = {
      owing(said, how) {
        asked += 1
        return front.cards.owing(said, how)
      },
    }
    const one = counting({ cards, failed: () => {} })

    const three = [one.count(), one.count(), one.count()]
    expect(asked).toBe(1)

    front.says(listing(vault('01A')))
    front.says(count(vault('01A')))
    front.ends()
    await Promise.all(three)
    expect(one.vaults.value).toHaveLength(1)

    // And the next ask is a count of its own, because the vaults have moved on.
    void one.count()
    expect(asked).toBe(2)
  })

  // A count that has already been worked out is what the window keeps showing
  // while the next one runs.
  it('leaves a vault at its last count while it is being counted again', async () => {
    const front = feeding()
    const one = counting({ cards: front.cards, failed: () => {} })

    void one.count()
    front.says(listing(vault('01A')))
    front.says(count(vault('01A', { due: 9, new: 0 })))
    front.ends()
    await settles()

    void one.count()
    front.says(listing(vault('01A')))
    await settles()

    expect(one.vaults.value[0]).toMatchObject({ counted: true, due: 9 })
  })

  it('says what went wrong and stops counting', async () => {
    const trouble: unknown[] = []
    const cards: Counts = {
      owing: async function* (): AsyncGenerator<Counted> {
        throw new Error('no registry')
      },
    }
    const one = counting({ cards, failed: (why) => trouble.push(why) })

    await one.count()

    expect(trouble).toHaveLength(1)
    expect(one.counting.value).toBe(false)
    expect(one.vaults.value).toHaveLength(0)
  })

  // A person who has chosen their vault, or closed the window, is not waiting
  // for the rest of the counts and is not told that they stopped.
  it('stops the count without calling it a failure', async () => {
    const trouble: unknown[] = []
    const front = feeding()
    const one = counting({ cards: front.cards, failed: (why) => trouble.push(why) })

    const asked = one.count()
    front.says(listing(vault('01A'), vault('01B')))
    await settles()
    one.stop()
    await asked

    expect(trouble).toHaveLength(0)
    expect(one.counting.value).toBe(false)
    expect(one.vaults.value).toHaveLength(2)
  })
})
