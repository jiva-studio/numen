// @vitest-environment jsdom
/**
 * A person who goes in and comes back out finds the window as they opened it.
 *
 * What a screen took up is declared beside the screen and release when the
 * screen is left. Nothing here names those holdings: it walks everything the
 * window hands out and compares the round trip against the opening. So a
 * collaborator left un-forgotten fails this whether or not anybody remembered
 * to write a case for it.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, h, isRef } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMockNoticeHost } from '@numen/plugin-sdk/testing'
import { Code, ConnectError } from '@connectrpc/connect'

const wire = vi.hoisted(() => {
  /** A stream that stays open, so nothing the window follows ever ends. */
  async function* waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  }

  /** A stream that hands over what the test sends it, and stays open between. */
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
    isCountHeld: false,
    counts: [] as AbortSignal[],
    daysAsked: [] as { vault: string; from: string; to: string }[],
    presetsAsked: [] as unknown[],
    notesAsked: [] as unknown[],
    answered: [] as { mark: string }[],
    responses: [] as { mark: string }[],
    takenBack: [] as unknown[],
    scrolled: [] as boolean[],
    agent: async (): Promise<{ unreachable: string }> => ({ unreachable: '' }),
    opened: { unwritten: [] as string[], skipped: 0 },
    isStartRefused: false,
    sessionsAsked: [] as unknown[],
  }
})

/** One vault owing something, so every screen has something to hold. */
const VAULT = {
  id: 'roots',
  name: 'Roots',
  path: '/vaults/Roots',
  faces: 2,
  due: 2,
  new: 0,
  decks: [{ deck: 'decks/Suffixes.md', faces: 2, due: 2, new: 0, learned: 0, unbegun: 0 }],
  presets: [],
  unread: '',
  reading: false,
}

vi.mock('../shared/clients', async (original) => ({
  ...(await original<typeof import('../shared/clients')>()),
  cards: {
    watchCardsDue: async function* (_: unknown, how?: { signal?: AbortSignal }) {
      const signal = how?.signal ?? new AbortController().signal
      wire.counts.push(signal)
      yield { day: '2026-04-02', vaults: [VAULT] }
      yield { day: '', vaults: [], counted: VAULT }
      if (!wire.isCountHeld) return
      await new Promise<never>((_, fail) => {
        signal.addEventListener('abort', () => fail(new Error('the count was ended')))
      })
    },
    watchReloads: () => wire.reloads.read(),
    listReviewDays: async (request: { vault: string; from: string; to: string }) => {
      wire.daysAsked.push(request)
      return {
        days: [
          {
            day: '2026-04-01',
            answered: 3,
            again: 1,
            hard: 0,
            good: 2,
            easy: 0,
            asked: 3,
            recalled: 2,
          },
        ],
        due: [{ day: '2026-04-03', answered: 1 }],
        streak: 4,
        answered: 3,
      }
    },
    getVaultDeckPreset: async (request: unknown) => {
      wire.presetsAsked.push(request)
      return { preset: undefined }
    },
    getDeckNeighbourhood: async (request: unknown) => {
      wire.notesAsked.push(request)
      return {
        notes: [
          {
            written: 'Prefixes',
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
      wire.sessionsAsked.push(request)
      if (wire.isStartRefused) {
        throw new ConnectError('the vault is closing', Code.FailedPrecondition)
      }
      return {
        run: 'run-1',
        cards: [
          {
            deck: 'decks/Suffixes.md',
            section: 'Suffixes',
            mark: 'mark-1',
            face: 'front',
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
    answerCard: async (request: { mark: string }) => {
      wire.responses.push(request)
      return { answer: 'answer-1' }
    },
    takeBackAnswer: async (request: unknown) => {
      wire.takenBack.push(request)
      return {}
    },
  },
  // Whether a card can be asked about is the agent's to answer, and the window
  // asks it as it opens.
  agentService: { getAgentState: () => wire.agent() },
  windowService: { watchTasks: () => wire.waits() },
}))

const { useWindow } = await import('./window')

type Window = ReturnType<typeof useWindow>

/** The corner the window last opened says its notices in. */
let notices = createMockNoticeHost()

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

/**
 * Everything the window hands out, as plain values: a ref stands for what it
 * holds, and what cannot be compared — the functions — stands for nothing.
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

/** What the reading is drawn in, as far as the keys reach into it. */
const Page = defineComponent({
  setup(_, { expose }) {
    expose({ scrollPage: (back: boolean) => wire.scrolled.push(back) })
    return () => null
  },
})

/** The window made, in a page, with the counts on the screen. */
const openWindow = async ({ isPageDrawn = true } = {}) => {
  notices = createMockNoticeHost()
  let target: Window | null = null
  const page = mount(
    defineComponent({
      setup() {
        target = useWindow(notices)
        return () => (isPageDrawn ? h(Page, { ref: 'page' }) : null)
      },
    }),
  )
  pages.push(page)
  await settle()
  await settle()
  return target as unknown as Window
}

/** The window taken into the session over the vault's one deck. */
const openSession = async () => {
  const sessionWindow = await openWindow()
  sessionWindow.vaults.choose('roots')
  await settle()
  await sessionWindow.decks.start('decks/Suffixes.md')
  return sessionWindow
}

/** A key pressed anywhere in the window, as the page then holds it. */
const press = async (key: string) => {
  const event = new KeyboardEvent('keydown', { key, cancelable: true })
  window.dispatchEvent(event)
  await settle()
  await settle()
  return event
}

/** The screen and the page drawn at the given widths, in layout pixels. */
const setWidths = (screen: number, page: number) => {
  Object.defineProperty(window.screen, 'width', { configurable: true, value: screen })
  Object.defineProperty(document.documentElement, 'clientWidth', {
    configurable: true,
    value: page,
  })
}

const pages: { unmount(): void }[] = []

beforeEach(() => {
  wire.reloads = wire.createChannel()
  wire.isCountHeld = false
  wire.counts.length = 0
  wire.daysAsked.length = 0
  wire.presetsAsked.length = 0
  wire.notesAsked.length = 0
  wire.responses.length = 0
  wire.takenBack.length = 0
  wire.scrolled.length = 0
  wire.agent = async () => ({ unreachable: '' })
  wire.opened = { unwritten: [], skipped: 0 }
  wire.isStartRefused = false
  wire.sessionsAsked.length = 0
})

afterEach(() => {
  for (const page of pages.splice(0)) page.unmount()
  Reflect.deleteProperty(window.screen, 'width')
  Reflect.deleteProperty(document.documentElement, 'clientWidth')
})

describe('a person who went in and came back out', () => {
  it('finds the window as they opened it', async () => {
    const window = await openWindow()
    const asOpened = shot(window)

    window.vaults.choose('roots')
    await settle()
    await window.decks.start('decks/Suffixes.md')
    window.session.toggleAgent()
    window.session.toggleNotes()
    window.session.state.show()
    await settle()
    await settle()

    await window.session.leave()
    await window.decks.goToVaults()
    await settle()
    await settle()

    expect(shot(window)).toStrictEqual(asOpened)
  })

  it('was holding something on the way', async () => {
    const window = await openWindow()

    window.vaults.choose('roots')
    await settle()
    await window.decks.start('decks/Suffixes.md')
    window.session.toggleAgent()
    window.session.toggleNotes()
    await settle()

    expect(window.on.value).toBe('session')
    expect(window.session.state.card.value?.mark).toBe('mark-1')
    expect(window.decks.reviewDays.streak.value).toBe(4)
    expect(window.session.agentPanel.about.value).not.toBeNull()
    expect(window.session.notesPanel.notes.value).toHaveLength(1)
  })
})

describe('a vault chosen', () => {
  it('ends the counting of the vaults', async () => {
    wire.isCountHeld = true
    const sessionWindow = await openWindow()
    expect(wire.counts[0]!.aborted).toBe(false)

    sessionWindow.vaults.choose('roots')

    expect(wire.counts[0]!.aborted).toBe(true)
  })

  it('asks which preset schedules each of its decks', async () => {
    const sessionWindow = await openWindow()
    sessionWindow.vaults.choose('roots')
    await settle()

    expect(wire.presetsAsked).toEqual([{ vault: 'roots', deck: 'decks/Suffixes.md' }])
  })

  it('asks for the days the wider of the screen and the page would draw', async () => {
    const askDays = async (screen: number, page: number) => {
      setWidths(screen, page)
      const sessionWindow = await openWindow()
      sessionWindow.vaults.choose('roots')
      await settle()
      const { from, to } = wire.daysAsked.at(-1)!
      return { from, to }
    }

    const wideDays = await askDays(2000, 2000)
    expect(await askDays(400, 2000)).toEqual(wideDays)
    expect(await askDays(2000, 400)).toEqual(wideDays)
    expect(await askDays(400, 400)).not.toEqual(wideDays)
  })
})

describe('a deck written underneath the window', () => {
  it('reads the days of the open vault again, and of no vault while none is open', async () => {
    const sessionWindow = await openWindow()

    wire.reloads.send({ reload: true })
    await settle()
    await settle()
    expect(wire.daysAsked).toHaveLength(0)

    sessionWindow.vaults.choose('roots')
    await settle()
    wire.reloads.send({ reload: true })
    await settle()
    await settle()
    expect(wire.daysAsked.map((one) => one.vault)).toEqual(['roots', 'roots'])

    await sessionWindow.decks.goToVaults()
    wire.reloads.send({ reload: true })
    await settle()
    await settle()
    expect(wire.daysAsked).toHaveLength(2)
  })
})

describe('a session sat down to', () => {
  it('says once what it could not act on', async () => {
    wire.opened = { unwritten: ['decks/Suffixes.md'], skipped: 2 }
    await openSession()

    const response = notices.getShown()
    expect(response).toHaveLength(2)
    expect(response.every((one) => one.tone === 'caution')).toBe(true)
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
    const sessionWindow = await openSession()

    expect(sessionWindow.on.value).toBe('session')
    expect(notices.getShown()).toHaveLength(0)
  })

  it('reviews all preset decks and reports errors', async () => {
    wire.opened = { unwritten: ['decks/Suffixes.md'], skipped: 0 }
    const sessionWindow = await openWindow()
    sessionWindow.vaults.choose('roots')
    await settle()

    await sessionWindow.decks.startPreset('presets/Daily.md')

    expect(sessionWindow.on.value).toBe('session')
    expect(wire.sessionsAsked).toEqual([{ vault: 'roots', deck: '', preset: 'presets/Daily.md' }])
    expect(notices.getShown().map((one) => one.tone)).toEqual(['caution'])
  })

  it('stays on the decks when a preset’s session cannot be opened', async () => {
    wire.isStartRefused = true
    const sessionWindow = await openWindow()
    sessionWindow.vaults.choose('roots')
    await settle()

    await sessionWindow.decks.startPreset('presets/Daily.md')

    expect(sessionWindow.on.value).toBe('decks')
    expect(notices.getShown().map((one) => one.tone)).toEqual(['alarm'])
  })

  it('ends the conversation with the card it was about', async () => {
    const sessionWindow = await openSession()
    sessionWindow.session.toggleAgent()
    expect(sessionWindow.session.agentPanel.about.value).not.toBeNull()

    sessionWindow.session.state.show()
    await sessionWindow.session.answerCard('good')

    expect(sessionWindow.session.agentPanel.about.value).toBeNull()
    expect(sessionWindow.session.agentPanel.open.value).toBe(false)
  })

  it('is left alone by a deck written underneath it', async () => {
    const sessionWindow = await openSession()
    const counted = wire.counts.length
    const daysRead = wire.daysAsked.length

    wire.reloads.send({ reload: true })
    await settle()
    await settle()

    expect(sessionWindow.on.value).toBe('session')
    expect(wire.counts).toHaveLength(counted)
    expect(wire.daysAsked).toHaveLength(daysRead)
  })

  it('stays on the decks when the session cannot be opened', async () => {
    wire.isStartRefused = true
    const sessionWindow = await openSession()

    expect(sessionWindow.on.value).toBe('decks')
    expect(notices.getShown().map((one) => one.tone)).toEqual(['alarm'])
  })

  it('reads the notes of the vault it is on', async () => {
    const sessionWindow = await openSession()
    sessionWindow.session.toggleNotes()
    await settle()

    expect(wire.notesAsked).toEqual([{ vault: 'roots', deck: 'decks/Suffixes.md' }])
  })

  it('lets the conversation in before the agent has returned whether it can be reached', async () => {
    wire.agent = () => new Promise<never>(() => {})
    const sessionWindow = await openSession()
    sessionWindow.session.toggleAgent()

    expect(sessionWindow.session.agentPanel.open.value).toBe(true)
    expect(notices.getShown()).toHaveLength(0)
  })

  it('keeps the conversation out when the agent cannot be reached, and says why', async () => {
    wire.agent = async () => ({ unreachable: 'no model is set up' })
    const sessionWindow = await openSession()
    sessionWindow.session.toggleAgent()

    expect(sessionWindow.session.agentPanel.open.value).toBe(false)
    expect(notices.getShown()).toMatchObject([{ text: 'no model is set up', tone: 'caution' }])
  })
})

describe('the keys of the window', () => {
  it('turn the card over, answer it and take the answer back', async () => {
    const sessionWindow = await openSession()

    expect((await press(' ')).defaultPrevented).toBe(true)
    expect(sessionWindow.session.state.isAnswerShown.value).toBe(true)

    // With the answer showing, the space bar asks nothing of the card and is
    // left to the page.
    expect((await press(' ')).defaultPrevented).toBe(false)

    await press('3')
    expect(wire.responses).toMatchObject([{ mark: 'mark-1' }])

    await press('u')
    expect(wire.takenBack).toHaveLength(1)
    expect(sessionWindow.session.state.card.value?.mark).toBe('mark-1')
  })

  it('scroll nothing, and break nothing, while no reading is drawn', async () => {
    const sessionWindow = await openWindow({ isPageDrawn: false })
    sessionWindow.vaults.choose('roots')
    await settle()
    await sessionWindow.decks.start('decks/Suffixes.md')
    sessionWindow.session.toggleNotes()
    await settle()
    const faults: unknown[] = []
    const onFault = (event: ErrorEvent) => faults.push(event.error)
    window.addEventListener('error', onFault)

    await press(' ')
    window.removeEventListener('error', onFault)

    expect(faults).toEqual([])
    expect(wire.scrolled).toEqual([])
  })

  it('scroll the reading with the space bar while it is up', async () => {
    const sessionWindow = await openSession()
    sessionWindow.session.toggleNotes()
    await settle()

    await press(' ')

    expect(wire.scrolled).toEqual([false])
    expect(sessionWindow.session.state.isAnswerShown.value).toBe(false)
  })

  it('leave the session for the decks counted again, and the decks for the vaults', async () => {
    const sessionWindow = await openSession()
    const counted = wire.counts.length
    const daysRead = wire.daysAsked.length

    await press('Escape')
    expect(sessionWindow.on.value).toBe('decks')
    expect(wire.counts).toHaveLength(counted + 1)
    expect(wire.daysAsked).toHaveLength(daysRead + 1)

    await press('Escape')
    expect(sessionWindow.on.value).toBe('vaults')
  })
})
