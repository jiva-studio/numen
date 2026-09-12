// @vitest-environment jsdom
/**
 * The vaults picked from the keyboard, in the window itself.
 *
 * The letter is drawn on the row by the screen and pressed at the window, and
 * only the window can say which vault it opened.
 */
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { counted, started, waits } = vi.hoisted(() => ({
  /**
   * What the front door answers: one vault owing nothing, and one owing on the
   * first of its two decks.
   */
  counted: {
    day: '2026-08-31',
    vaults: [
      {
        id: 'physics',
        name: 'Physics',
        path: '/vaults/Physics',
        faces: 2,
        due: 0,
        new: 0,
        decks: [{ deck: 'decks/Heat.md', faces: 2, due: 0, new: 0 }],
        presets: [],
        unread: '',
        reading: false,
      },
      {
        id: 'words',
        name: 'Words',
        path: '/vaults/Words',
        faces: 4,
        due: 2,
        new: 0,
        decks: [
          { deck: 'decks/Words.md', faces: 2, due: 2, new: 0 },
          { deck: 'decks/Roots.md', faces: 2, due: 0, new: 0 },
        ],
        presets: [],
        unread: '',
        reading: false,
      },
    ],
  },
  /** Every session the window opened, by what it was opened over. */
  started: [] as { deck: string }[],
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

vi.mock('@/shared/clients', async (original) => ({
  ...(await original<typeof import('@/shared/clients')>()),
  cards: {
    // The vaults, then each of their counts, the way the front door answers.
    watchCardsDue: async function* () {
      yield { day: counted.day, vaults: counted.vaults }
      for (const one of counted.vaults) yield { day: '', vaults: [], counted: one }
    },
    watchReloads: () => waits(),
    getAgentState: async () => ({ unreachable: '' }),
    listReviewDays: async () => ({ days: [], due: [], streak: 0, answered: 0 }),
    getVaultDeckPreset: async () => ({ preset: undefined }),
    startSession: async (said: { deck: string }) => {
      started.push(said)
      return { run: 'run', asked: [], unwritten: [], skipped: 0 }
    },
  },
  itself: { watchTasks: () => waits() },
}))

const { default: App } = await import('./App.vue')
const { Decks } = await import('@/pages/decks')

/** The window drawn, with the vaults counted and on the screen. */
const mountWindow = async () => {
  const window = mount(App, { attachTo: document.body })
  windows.push(window)
  await settle()
  await settle()
  return window
}

const settle = () => new Promise((done) => setTimeout(done, 0))

/**
 * Every window a test drew. A window listens for keystrokes for as long as it
 * is mounted, and the next test draws its own.
 */
const windows: VueWrapper[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
  started.splice(0)
})

/** One keystroke, taken on the window as a person takes it. */
const press = async (key: string, more: KeyboardEventInit = {}) => {
  globalThis.dispatchEvent(new KeyboardEvent('keydown', { key, cancelable: true, ...more }))
  await settle()
  await settle()
}

/** The vault the window went into, and nothing while it is still on the list. */
const getOpenVault = (window: VueWrapper): string => {
  const decks = window.findComponent(Decks)
  return decks.exists() ? (decks.props('vault') as { vault: string }).vault : ''
}

describe('a letter pressed on the vaults', () => {
  it('opens the vault standing at it', async () => {
    const window = await mountWindow()

    await press('b')

    expect(getOpenVault(window)).toBe('words')
  })

  it('opens the first of them with the first letter of the alphabet', async () => {
    const window = await mountWindow()

    await press('a')

    expect(getOpenVault(window)).toBe('physics')
  })

  it('opens nothing where no vault stands at the letter', async () => {
    const window = await mountWindow()

    await press('c')

    expect(getOpenVault(window)).toBe('')
  })

  it('opens nothing where the letter is held with the overlay key', async () => {
    const window = await mountWindow()

    await press('a', { ctrlKey: true })

    expect(getOpenVault(window)).toBe('')
  })
})

// The letter drawn on a row and the row itself are one act, so a row that
// cannot be pressed is a letter that does nothing. A session opened over a deck
// owing nothing is an empty session, and it mints marks in the vault to hold it.
describe('a letter pressed on the decks', () => {
  /** The window on the decks of the vault standing at this letter. */
  const on = async (vault: string) => {
    const window = await mountWindow()
    await press(vault)
    return window
  }

  it('sits down to the deck standing at it', async () => {
    await on('b')

    await press('a')

    expect(started).toStrictEqual([{ vault: 'words', deck: 'decks/Words.md' }])
  })

  it('sits down to nothing where that deck owes nothing', async () => {
    await on('b')

    await press('b')

    expect(started).toStrictEqual([])
  })

  it('sits down to nothing where the whole vault owes nothing', async () => {
    await on('a')

    await press('Enter')

    expect(started).toStrictEqual([])
  })

  it('sits down to the whole vault where it owes something', async () => {
    await on('b')

    await press('Enter')

    expect(started).toStrictEqual([{ vault: 'words', deck: '' }])
  })
})
