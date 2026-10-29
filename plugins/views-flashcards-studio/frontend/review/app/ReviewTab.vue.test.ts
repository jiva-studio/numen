// @vitest-environment jsdom
/**
 * The review as the window shows it: the decks once the vault is counted, and
 * the letters a deck is picked by.
 */
import { flushPromises, mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import { effectScope } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMockAgentHost, createMockNoticeHost } from '@numen/plugin-sdk/testing'

const { counted, started, waits } = vi.hoisted(() => ({
  counted: {
    day: '2026-08-31',
    vaults: [
      {
        id: 'physics',
        name: 'Physics',
        path: '/vaults/Physics',
        reviewCards: 2,
        due: 0,
        new: 0,
        decks: [{ deck: 'decks/Heat.md', reviewCards: 2, due: 0, new: 0, learned: 0, unbegun: 0 }],
        presets: [],
        unread: '',
        isIndexing: false,
      },
      {
        id: 'words',
        name: 'Words',
        path: '/vaults/Words',
        reviewCards: 4,
        due: 2,
        new: 0,
        decks: [
          { deck: 'decks/Words.md', reviewCards: 2, due: 2, new: 0, learned: 0, unbegun: 0 },
          { deck: 'decks/Roots.md', reviewCards: 2, due: 0, new: 0, learned: 0, unbegun: 0 },
        ],
        presets: [],
        unread: '',
        isIndexing: false,
      },
      {
        id: 'gone',
        name: 'Gone',
        path: '/vaults/Gone',
        reviewCards: 0,
        due: 0,
        new: 0,
        decks: [],
        presets: [],
        unread: 'the folder has gone',
        isIndexing: false,
      },
    ],
    /** The vaults the count names after the list, by identity. */
    named: ['physics', 'words'],
  },
  /** Every session the review opened, by what it was opened over. */
  started: [] as { deck: string }[],
  /** A stream that stays open, so nothing the review follows ever ends. */
  async *waits(): AsyncGenerator<never> {
    await new Promise<never>(() => {})
  },
}))

vi.mock('../shared/clients', async (original) => ({
  ...(await original<typeof import('../shared/clients')>()),
  cards: {
    watchCardsDue: async function* () {
      yield { day: counted.day, vaults: counted.vaults }
      for (const one of counted.vaults) {
        if (counted.named.includes(one.id)) yield { day: '', vaults: [], counted: one }
      }
    },
    watchReloads: () => waits(),
    listReviewDays: async () => ({ days: [], due: [], streak: 0, reviewed: 0 }),
    getVaultDeckPreset: async () => ({ preset: undefined }),
    startSession: async (response: { deck: string }) => {
      started.push(response)
      return { run: 'run', cards: [], unwritten: [], skipped: 0 }
    },
  },
}))

const { default: ReviewTab } = await import('./ReviewTab.vue')
const { useReview } = await import('./review')
const { createVaultCounts } = await import('./counts')
const { mountNothing } = await import('../shared/views')
const { Decks } = await import('../pages/decks')

type Review = ReturnType<typeof useReview>

const closers: (() => void)[] = []

/** The review of a vault, drawn as the window draws it. */
const mountReview = async (vault: string) => {
  const counts = createVaultCounts({ reportError: vi.fn() })
  const scope = effectScope()
  const state = scope.run(() =>
    useReview({
      vault,
      counts,
      notices: createMockNoticeHost(),
      agent: createMockAgentHost(),
      mountView: mountNothing,
      goToVaults: () => {},
    }),
  ) as Review
  const tab: VueWrapper = mount(ReviewTab, { props: { state } })
  closers.push(
    () => tab.unmount(),
    () => scope.stop(),
    counts.dispose,
  )
  await flushPromises()
  await flushPromises()
  return { tab, state }
}

afterEach(() => {
  for (const close of closers.splice(0)) close()
  started.splice(0)
})

/** One keystroke, as the window hands it to the review in front. */
const press = async (state: Review, key: string) => {
  state.onKeyDown(new KeyboardEvent('keydown', { key, cancelable: true }))
  await flushPromises()
}

describe('the review of a counted vault', () => {
  it('draws the decks of that vault', async () => {
    const { tab } = await mountReview('words')

    const decks = tab.findComponent(Decks)
    expect((decks.props('vault') as { vault: string }).vault).toBe('words')
  })
})

describe('the review of a vault whose count is not in', () => {
  it('says it is counting, and draws no decks', async () => {
    const { tab } = await mountReview('unknown')

    expect(tab.findComponent(Decks).exists()).toBe(false)
    expect(tab.find('[role="status"]').text()).toContain('Counting the cards')
  })

  it('says why the vault could not be counted', async () => {
    const { tab } = await mountReview('gone')

    expect(tab.findComponent(Decks).exists()).toBe(false)
    expect(tab.find('.review__waiting').text()).toBe('the folder has gone')
  })
})

// The letter drawn on a row and the row itself are one act, so a row that
// cannot be pressed is a letter that does nothing. A session opened over a deck
// owing nothing is an empty session, and it mints card ids in the vault to hold it.
describe('a letter pressed on the decks', () => {
  it('opens review for target deck', async () => {
    const { state } = await mountReview('words')

    await press(state, 'a')

    expect(started).toStrictEqual([{ vault: 'words', deck: 'decks/Words.md' }])
  })

  it('does not start review when deck has no due cards', async () => {
    const { state } = await mountReview('words')

    await press(state, 'b')

    expect(started).toStrictEqual([])
  })

  it('does not start review when vault has no due cards', async () => {
    const { state } = await mountReview('physics')

    await press(state, 'Enter')

    expect(started).toStrictEqual([])
  })

  it('starts review for vault with due cards', async () => {
    const { state } = await mountReview('words')

    await press(state, 'Enter')

    expect(started).toStrictEqual([{ vault: 'words', deck: '' }])
  })
})
