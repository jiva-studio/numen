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

const { counted, waits } = vi.hoisted(() => ({
  /** What the front door answers: two vaults, each with a deck owing. */
  counted: {
    day: '2026-08-31',
    vaults: [
      {
        vaultId: 'physics',
        name: 'Physics',
        path: '/vaults/Physics',
        faces: 2,
        due: 1,
        new: 1,
        decks: [{ deck: 'decks/Heat.md', faces: 2, due: 1, new: 1 }],
        presets: [],
        unread: '',
      },
      {
        vaultId: 'words',
        name: 'Words',
        path: '/vaults/Words',
        faces: 2,
        due: 2,
        new: 0,
        decks: [{ deck: 'decks/Words.md', faces: 2, due: 2, new: 0 }],
        presets: [],
        unread: '',
      },
    ],
  },
  /** A stream that stays open, so nothing the window follows ever ends. */
  async *waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

vi.mock('./core', async (original) => ({
  ...(await original<typeof import('./core')>()),
  cards: {
    owing: async () => counted,
    moving: () => waits(),
    asking: async () => ({ unreachable: '' }),
    reviewed: async () => ({ days: [], due: [], streak: 0, answered: 0 }),
    scheduling: async () => ({ preset: undefined }),
  },
}))

const { default: App } = await import('./App.vue')
const Decks = (await import('./Decks.vue')).default

/** The window drawn, with the vaults counted and on the screen. */
const drawn = async () => {
  const window = mount(App, { attachTo: document.body })
  windows.push(window)
  await settles()
  await settles()
  return window
}

const settles = () => new Promise((done) => setTimeout(done, 0))

/**
 * Every window a test drew. A window listens for keystrokes for as long as it
 * is mounted, and the next test draws its own.
 */
const windows: VueWrapper[] = []

afterEach(() => {
  for (const window of windows.splice(0)) window.unmount()
})

/** One keystroke, taken on the window as a person takes it. */
const press = async (key: string, more: KeyboardEventInit = {}) => {
  globalThis.dispatchEvent(new KeyboardEvent('keydown', { key, cancelable: true, ...more }))
  await settles()
  await settles()
}

/** The vault the window went into, and nothing while it is still on the list. */
const opened = (window: VueWrapper): string => {
  const decks = window.findComponent(Decks)
  return decks.exists() ? (decks.props('vault') as { vaultId: string }).vaultId : ''
}

describe('a letter pressed on the vaults', () => {
  it('opens the vault standing at it', async () => {
    const window = await drawn()

    await press('b')

    expect(opened(window)).toBe('words')
  })

  it('opens the first of them with the first letter of the alphabet', async () => {
    const window = await drawn()

    await press('a')

    expect(opened(window)).toBe('physics')
  })

  it('opens nothing where no vault stands at the letter', async () => {
    const window = await drawn()

    await press('c')

    expect(opened(window)).toBe('')
  })

  it('opens nothing where the letter is held with the overlay key', async () => {
    const window = await drawn()

    await press('a', { ctrlKey: true })

    expect(opened(window)).toBe('')
  })
})
