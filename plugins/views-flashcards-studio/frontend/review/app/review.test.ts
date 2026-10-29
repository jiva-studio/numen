// @vitest-environment jsdom
/**
 * The review of one vault, as the window opens it and closes it.
 *
 * What a screen took up is declared beside the screen and released when the
 * screen is left. The round trip below names none of those holdings: it walks
 * everything the review hands out and compares it against the opening. So a
 * collaborator left un-forgotten fails it whether or not anybody remembered
 * to write a case for it.
 */
import { effectScope, isRef } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMockAgentHost, createMockNoticeHost } from '@numen/plugin-sdk/testing'
import { ApiFailure } from '../../api/failure'

const wire = vi.hoisted(() => {
  /** A stream that stays open, so nothing the window follows ever ends. */
  async function* waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  }

  /** A stream that delivers what the test sends it, and stays open between sends. */
  const createChannel = () => {
    const queue: unknown[] = []
    let wake: (() => void) | null = null
    return {
      send(item: unknown) {
        queue.push(item)
        wake?.()
        wake = null
      },
      async *read(): AsyncGenerator<unknown> {
        for (;;) {
          while (queue.length) yield queue.shift()
          await new Promise<void>((isDone) => {
            wake = isDone
          })
        }
      },
    }
  }

  return {
    waits,
    createChannel,
    reloads: createChannel(),
    /** Whether a count stays on its way until the window ends it. */
    isCountPending: false,
    /** Whether the count names the vault's own numbers, after the list. */
    isVaultCounted: true,
    counts: [] as AbortSignal[],
    requestedDays: [] as { vault: string; from: string; to: string }[],
    requestedPresets: [] as unknown[],
    requestedNotes: [] as unknown[],
    answered: [] as { cardId: string }[],
    responses: [] as { cardId: string }[],
    takenBack: [] as unknown[],
    scrolled: [] as boolean[],
    opened: { unwritten: [] as string[], skipped: 0 },
    isStartRefused: false,
    requestedSessions: [] as unknown[],
  }
})

/** One vault owing something, so every screen has something to hold. */
const VAULT = {
  id: 'roots',
  name: 'Roots',
  path: '/vaults/Roots',
  reviewCards: 2,
  due: 2,
  new: 0,
  decks: [{ deck: 'decks/Suffixes.md', reviewCards: 2, due: 2, new: 0, learned: 0, unbegun: 0 }],
  presets: [],
  unread: '',
  isIndexing: false,
}

vi.mock('../shared/clients', async (original) => ({
  ...(await original<typeof import('../shared/clients')>()),
  cards: {
    watchCardsDue: async function* (_: unknown, how?: { signal?: AbortSignal }) {
      const signal = how?.signal ?? new AbortController().signal
      wire.counts.push(signal)
      yield { day: '2026-04-02', vaults: [VAULT] }
      if (wire.isVaultCounted) yield { day: '', vaults: [], counted: VAULT }
      if (!wire.isCountPending) return
      await new Promise<never>((_, fail) => {
        signal.addEventListener('abort', () => fail(new Error('the count was ended')))
      })
    },
    watchReloads: () => wire.reloads.read(),
    listReviewDays: async (request: { vault: string; from: string; to: string }) => {
      wire.requestedDays.push(request)
      return {
        days: [
          {
            day: '2026-04-01',
            reviewed: 3,
            again: 1,
            hard: 0,
            good: 2,
            easy: 0,
            tested: 3,
            recalled: 2,
          },
        ],
        due: [{ day: '2026-04-03', reviewed: 1 }],
        streak: 4,
        reviewed: 3,
      }
    },
    getVaultDeckPreset: async (request: unknown) => {
      wire.requestedPresets.push(request)
      return { preset: undefined }
    },
    getDeckNeighbourhood: async (request: unknown) => {
      wire.requestedNotes.push(request)
      return {
        notes: [
          {
            address: 'Prefixes',
            path: 'notes/Prefixes.md',
            title: 'Prefixes',
            body: 'A word before the word.',
            label: '',
            hasPoints: true,
            isAmbiguous: false,
          },
        ],
        unread: 0,
      }
    },
    startSession: async (request: unknown) => {
      wire.requestedSessions.push(request)
      if (wire.isStartRefused) {
        throw new ApiFailure({ code: 'failed_precondition', message: 'the vault is closing' })
      }
      return {
        run: 'run-1',
        cards: [
          {
            deck: 'decks/Suffixes.md',
            section: 'Suffixes',
            cardId: 'mark-1',
            template: 'front',
            heading: '-ness',
            front: '-ness',
            back: 'the state of being',
            seen: false,
            intervals: undefined,
          },
        ],
        ...wire.opened,
      }
    },
    answerCard: async (request: { cardId: string }) => {
      wire.responses.push(request)
      return { answer: 'answer-1' }
    },
    takeBackAnswer: async (request: unknown) => {
      wire.takenBack.push(request)
      return {}
    },
  },
}))

const { useReview } = await import('./review')
const { createVaultCounts } = await import('./counts')
const { mountNothing } = await import('../shared/views')

type Review = ReturnType<typeof useReview>

/** The notice host of the most recently opened review. */
let notices = createMockNoticeHost()

/** The agent of the window the review last opened. */
let agent = createMockAgentHost()

/** The callback the most recently opened review calls to return to the vaults. */
let goToVaults = vi.fn()

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/** What ends each review a test opened, and the counting under it. */
const closers: (() => void)[] = []

/** The review opened on the vault, with the counts in, as the window opens it. */
const openReview = async ({ isPageDrawn = true } = {}) => {
  notices = createMockNoticeHost()
  goToVaults = vi.fn()
  const counts = createVaultCounts({ reportError: vi.fn() })
  const scope = effectScope()
  const review = scope.run(() =>
    useReview({ vault: 'roots', counts, notices, agent, mountView: mountNothing, goToVaults }),
  ) as Review
  if (isPageDrawn) {
    review.notesPage.value = { scrollPage: (back: boolean) => wire.scrolled.push(back) }
  }
  const close = () => scope.stop()
  closers.push(close, counts.dispose)
  await settle()
  await settle()
  return { review, close }
}

/** The review taken into the session over the vault's one deck. */
const openSession = async () => {
  const result = await openReview()
  await result.review.decks.start('decks/Suffixes.md')
  return result.review
}

/**
 * Everything the review exposes, as plain values: a ref becomes the value it
 * contains, and functions, which cannot be compared, become null.
 */
const shot = (value: unknown): unknown => {
  if (isRef(value)) return shot(value.value)
  if (typeof value === 'function') return null
  if (value instanceof Map) return [...value].map(([key, val]) => [key, shot(val)])
  if (Array.isArray(value)) return value.map(shot)
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([key, val]) => [key, shot(val)]))
  }
  return value
}

/** A key pressed while the review is focused, as the host delivers it. */
const press = async (review: Review, key: string) => {
  const event = new KeyboardEvent('keydown', { key, cancelable: true })
  const isKept = review.onKeyDown(event)
  await settle()
  await settle()
  return { event, isKept }
}

/** The screen and the page drawn at the given widths, in layout pixels. */
const setWidths = (screen: number, page: number) => {
  Object.defineProperty(window.screen, 'width', { configurable: true, value: screen })
  Object.defineProperty(document.documentElement, 'clientWidth', {
    configurable: true,
    value: page,
  })
}

beforeEach(() => {
  wire.reloads = wire.createChannel()
  wire.isCountPending = false
  wire.isVaultCounted = true
  wire.counts.length = 0
  wire.requestedDays.length = 0
  wire.requestedPresets.length = 0
  wire.requestedNotes.length = 0
  wire.responses.length = 0
  wire.takenBack.length = 0
  wire.scrolled.length = 0
  agent = createMockAgentHost()
  wire.opened = { unwritten: [], skipped: 0 }
  wire.isStartRefused = false
  wire.requestedSessions.length = 0
})

afterEach(() => {
  for (const close of closers.splice(0)) close()
  Reflect.deleteProperty(window.screen, 'width')
  Reflect.deleteProperty(document.documentElement, 'clientWidth')
})

describe('a person who sat down and came back to the decks', () => {
  it('finds the review as it opened', async () => {
    const { review } = await openReview()
    const asOpened = shot(review)

    await review.decks.start('decks/Suffixes.md')
    review.session.toggleAgent()
    review.session.toggleNotes()
    review.session.state.show()
    await settle()
    await settle()

    await review.session.leave()
    await settle()
    await settle()

    expect(shot(review)).toStrictEqual(asOpened)
  })

  it('was holding something on the way', async () => {
    const review = await openSession()
    review.session.toggleAgent()
    review.session.toggleNotes()
    await settle()

    expect(review.on.value).toBe('session')
    expect(review.session.state.card.value?.cardId).toBe('mark-1')
    expect(review.decks.reviewDays.streak.value).toBe(4)
    expect(review.session.agentPanel.about.value).not.toBeNull()
    expect(review.session.notesPanel.notes.value).toHaveLength(1)
  })
})

describe('a review opened on a vault', () => {
  it('ends the counting of the vaults once its own is counted', async () => {
    wire.isCountPending = true
    await openReview()

    expect(wire.counts).toHaveLength(1)
    expect(wire.counts[0]!.aborted).toBe(true)
  })

  it('asks which preset schedules each of its decks', async () => {
    await openReview()

    expect(wire.requestedPresets).toEqual([{ vault: 'roots', deck: 'decks/Suffixes.md' }])
  })

  it('waits for its vault to be counted, and keeps the counting going until it is', async () => {
    wire.isCountPending = true
    wire.isVaultCounted = false
    const { review } = await openReview()

    expect(review.vaultCount.value?.isCounted).toBe(false)
    expect(wire.requestedPresets).toEqual([])
    expect(wire.counts[0]!.aborted).toBe(false)
    expect(wire.requestedDays.map((one) => one.vault)).toEqual(['roots'])
  })

  it('asks for the days the wider of the screen and the page would draw', async () => {
    const askDays = async (screen: number, page: number) => {
      setWidths(screen, page)
      await openReview()
      const { from, to } = wire.requestedDays.at(-1)!
      return { from, to }
    }

    const wideDays = await askDays(2000, 2000)
    expect(await askDays(400, 2000)).toEqual(wideDays)
    expect(await askDays(2000, 400)).toEqual(wideDays)
    expect(await askDays(400, 400)).not.toEqual(wideDays)
  })

  it('goes back to the vaults counting them again', async () => {
    const { review } = await openReview()
    const counted = wire.counts.length

    review.decks.goToVaults()
    await settle()

    expect(goToVaults).toHaveBeenCalledOnce()
    expect(wire.counts).toHaveLength(counted + 1)
  })
})

describe('a deck written underneath the review', () => {
  it('reads the days of its vault again, and nothing once the review is closed', async () => {
    const { close } = await openReview()

    wire.reloads.send({ reload: true })
    await settle()
    await settle()
    expect(wire.requestedDays.map((one) => one.vault)).toEqual(['roots', 'roots'])

    close()
    wire.reloads.send({ reload: true })
    await settle()
    await settle()
    expect(wire.requestedDays).toHaveLength(2)
  })
})

describe('a review closed by the window', () => {
  it('lets go of the session it was in', async () => {
    const review = await openSession()
    review.session.toggleAgent()
    expect(review.session.agentPanel.about.value).not.toBeNull()

    for (const close of closers.splice(0)) close()

    expect(review.on.value).toBe('decks')
    expect(review.session.state.card.value).toBeNull()
    expect(review.session.agentPanel.about.value).toBeNull()
  })
})

describe('a session sat down to', () => {
  it('says once what it could not act on', async () => {
    wire.opened = { unwritten: ['decks/Suffixes.md'], skipped: 2 }
    await openSession()

    const response = notices.getShown()
    expect(response).toHaveLength(2)
    expect(response.every((one) => one.severity === 'warning')).toBe(true)
    expect(response[0]!.text).toContain('Suffixes')
    expect(response[1]!.text).toContain('2')
  })

  it('names every deck it could not act on, in one line', async () => {
    wire.opened = { unwritten: ['decks/Suffixes.md', 'decks/Prefixes.md'], skipped: 0 }
    await openSession()

    expect(notices.getShown().map((one) => one.text)).toEqual([
      'Not asked from Suffixes, Prefixes: the deck could not be written.',
    ])
  })

  it('says nothing when it could act on everything', async () => {
    const review = await openSession()

    expect(review.on.value).toBe('session')
    expect(notices.getShown()).toHaveLength(0)
  })

  it('reviews all preset decks and reports errors', async () => {
    wire.opened = { unwritten: ['decks/Suffixes.md'], skipped: 0 }
    const { review } = await openReview()
    await settle()

    await review.decks.startPreset('presets/Daily.md')

    expect(review.on.value).toBe('session')
    expect(wire.requestedSessions).toEqual([
      { vault: 'roots', deck: '', preset: 'presets/Daily.md' },
    ])
    expect(notices.getShown().map((one) => one.severity)).toEqual(['warning'])
  })

  it('stays on the decks when a preset’s session cannot be opened', async () => {
    wire.isStartRefused = true
    const { review } = await openReview()
    await settle()

    await review.decks.startPreset('presets/Daily.md')

    expect(review.on.value).toBe('decks')
    expect(notices.getShown().map((one) => one.severity)).toEqual(['error'])
  })

  it('ends the conversation with the card it was about', async () => {
    const review = await openSession()
    review.session.toggleAgent()
    expect(review.session.agentPanel.about.value).not.toBeNull()

    review.session.state.show()
    await review.session.answerCard('good')

    expect(review.session.agentPanel.about.value).toBeNull()
    expect(review.session.agentPanel.open.value).toBe(false)
  })

  it('is left alone by a deck written underneath it', async () => {
    const review = await openSession()
    const counted = wire.counts.length
    const daysRead = wire.requestedDays.length

    wire.reloads.send({ reload: true })
    await settle()
    await settle()

    expect(review.on.value).toBe('session')
    expect(wire.counts).toHaveLength(counted)
    expect(wire.requestedDays).toHaveLength(daysRead)
  })

  it('stays on the decks when the session cannot be opened', async () => {
    wire.isStartRefused = true
    const review = await openSession()

    expect(review.on.value).toBe('decks')
    expect(notices.getShown().map((one) => one.severity)).toEqual(['error'])
  })

  it('reads the notes of the vault it is on', async () => {
    const review = await openSession()
    review.session.toggleNotes()
    await settle()

    expect(wire.requestedNotes).toEqual([{ vault: 'roots', deck: 'decks/Suffixes.md' }])
  })

  it('keeps the conversation out when the agent cannot be reached, and says why', async () => {
    agent.setUnreachable('no model is set up')
    const review = await openSession()
    review.session.toggleAgent()

    expect(review.session.agentPanel.open.value).toBe(false)
    expect(notices.getShown()).toMatchObject([{ text: 'no model is set up', severity: 'warning' }])
  })
})

describe('the keys of the review', () => {
  it('turn the card over, answer it and take the answer back', async () => {
    const review = await openSession()

    expect((await press(review, ' ')).isKept).toBe(true)
    expect(review.session.state.isAnswerShown.value).toBe(true)

    //     // With the answer displayed, the space bar does nothing for the card and is
    //     // left to the page.
    expect((await press(review, ' ')).isKept).toBe(false)

    await press(review, '3')
    expect(wire.responses).toMatchObject([{ cardId: 'mark-1' }])

    await press(review, 'u')
    expect(wire.takenBack).toHaveLength(1)
    expect(review.session.state.card.value?.cardId).toBe('mark-1')
  })

  it('scroll nothing, and break nothing, while no reading is drawn', async () => {
    const { review } = await openReview({ isPageDrawn: false })
    await settle()
    await review.decks.start('decks/Suffixes.md')
    review.session.toggleNotes()
    await settle()
    const faults: unknown[] = []
    const onFault = (event: ErrorEvent) => faults.push(event.error)
    window.addEventListener('error', onFault)

    await press(review, ' ')
    window.removeEventListener('error', onFault)

    expect(faults).toEqual([])
    expect(wire.scrolled).toEqual([])
  })

  it('scroll the reading with the space bar while it is up', async () => {
    const review = await openSession()
    review.session.toggleNotes()
    await settle()

    await press(review, ' ')

    expect(wire.scrolled).toEqual([false])
    expect(review.session.state.isAnswerShown.value).toBe(false)
  })

  it('leave the session for the decks counted again, and the decks for the vaults of the window', async () => {
    const review = await openSession()
    const counted = wire.counts.length
    const daysRead = wire.requestedDays.length

    await press(review, 'Escape')
    expect(review.on.value).toBe('decks')
    expect(wire.counts).toHaveLength(counted + 1)
    expect(wire.requestedDays).toHaveLength(daysRead + 1)

    await press(review, 'Escape')
    expect(goToVaults).toHaveBeenCalledOnce()
  })
})
