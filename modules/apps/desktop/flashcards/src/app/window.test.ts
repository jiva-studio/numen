// @vitest-environment jsdom
/**
 * A person who goes in and comes back out finds the window as they opened it.
 *
 * What a screen took up is declared beside the screen and let go of when the
 * screen is left. Nothing here names those holdings: it walks everything the
 * window hands out and compares the round trip against the opening. So a
 * collaborator left un-forgotten fails this whether or not anybody remembered
 * to write a case for it.
 */
import { mount } from '@vue/test-utils'
import { defineComponent, isRef } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { waits } = vi.hoisted(() => ({
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

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

vi.mock('@/shared/clients', async (original) => ({
  ...(await original<typeof import('@/shared/clients')>()),
  cards: {
    watchCardsDue: async function* () {
      yield { day: '2026-04-02', vaults: [VAULT] }
      yield { day: '', vaults: [], counted: VAULT }
    },
    watchReloads: () => waits(),
    listReviewDays: async () => ({
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
    }),
    getVaultDeckPreset: async () => ({ preset: undefined }),
    getDeckNeighbourhood: async () => ({
      notes: [
        {
          written: 'Prefixes',
          path: 'notes/Prefixes.md',
          title: 'Prefixes',
          body: 'A word before the word.',
          label: '',
          points: true,
          ambiguous: false,
        },
      ],
      unread: 0,
    }),
    startSession: async () => ({
      run: 'run-1',
      asked: [
        {
          deck: 'decks/Suffixes.md',
          section: 'Suffixes',
          mark: 'mark-1',
          face: 'front',
          heading: '-ness',
          front: '-ness',
          back: 'the state of being',
          seen: false,
          ahead: undefined,
        },
      ],
      unwritten: [],
      skipped: 0,
    }),
  },
  // Whether a card can be asked about is the agent's to answer, and the window
  // asks it as it opens.
  agentService: { getAgentState: async () => ({ unreachable: '' }) },
  itself: { watchTasks: () => waits() },
}))

const { useWindow } = await import('./window')

type Window = ReturnType<typeof useWindow>

const settle = () => new Promise((done) => setTimeout(done, 0))

/**
 * Everything the window hands out, as plain values: a ref stands for what it
 * holds, and what cannot be compared — the functions — stands for nothing.
 */
const shot = (value: unknown): unknown => {
  if (isRef(value)) return shot(value.value)
  if (typeof value === 'function') return null
  if (value instanceof Map) return [...value].map(([key, held]) => [key, shot(held)])
  if (Array.isArray(value)) return value.map(shot)
  if (value && typeof value === 'object') {
    return Object.fromEntries(Object.entries(value).map(([key, held]) => [key, shot(held)]))
  }
  return value
}

/** The window made, in a page, with the counts on the screen. */
const openWindow = async () => {
  let held: Window | null = null
  const page = mount(
    defineComponent({
      setup() {
        held = useWindow()
        return () => null
      },
    }),
  )
  pages.push(page)
  await settle()
  await settle()
  return held as unknown as Window
}

const pages: { unmount(): void }[] = []

afterEach(() => {
  for (const page of pages.splice(0)) page.unmount()
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
    expect(window.decks.done.streak.value).toBe(4)
    expect(window.session.agentPanel.about.value).not.toBeNull()
    expect(window.session.notesPanel.notes.value).toHaveLength(1)
  })
})
