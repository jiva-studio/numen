/**
 * The due counts of every vault the installation holds today.
 *
 * Kept outside the template because counting starts from three places — the
 * window opening, a session ending, and a vault changing — and one count runs
 * at a time however many request it.
 *
 * The vaults arrive first, with their name and location, and each count
 * follows separately. A vault whose count has not arrived is displayed from
 * the list alone, without a count.
 */
import { ref, shallowRef } from 'vue'
import type { StopReason } from '../../../../api/types.js'

import type { VaultCardsDue } from '../types'

export interface CardsDueClient {
  watchCardsDue(
    req: Record<string, never>,
    how?: { signal?: AbortSignal },
  ): AsyncIterable<DueCounts>
}

/** One vault, as the count answers about it. */
export interface VaultCounts {
  id: string
  name: string
  path: string
  reviewCards: number
  due: number
  new: number
  decks: readonly {
    deck: string
    reviewCards: number
    due: number
    new: number
    learned: number
    unbegun: number
  }[]
  presets: readonly {
    preset: string
    title: string
    decks: number
    cards: number
    owedDue: number
    owedNew: number
    reviewed: number
    reviewedNew: number
    reviewedReviews: number
    durationMs: number
    new: number
    reviews: number
    minutes: number
    closesNew: string
    closesReviews: string
    closesMinutes: string
    stopsOn: StopReason
  }[]
  unread: string
  isIndexing: boolean
}

/** One message of the count. */
export interface DueCounts {
  /** The review day these counts stand in, which begins at the hour the settings name. */
  day: string
  /** Every vault the installation holds, in the first message and in no other. */
  vaults: readonly VaultCounts[]
  /** One vault worked out, in every message after the first. */
  counted?: VaultCounts | undefined
}

/** A length of time as the application holds one, which is in minutes. */
const getMinutes = (ms: number): number => ms / 60000

export interface CountsDeps {
  cards: CardsDueClient
  reportError(why: unknown): void
}

const createUncountedVault = (one: VaultCounts): VaultCardsDue => ({
  vault: one.id,
  name: one.name,
  path: one.path,
  isCounted: false,
  reviewCards: 0,
  due: 0,
  new: 0,
  decks: [],
  presets: [],
  unread: one.unread,
  isIndexing: one.isIndexing,
})

const createCountedVault = (one: VaultCounts): VaultCardsDue => ({
  vault: one.id,
  name: one.name,
  path: one.path,
  isCounted: true,
  reviewCards: one.reviewCards,
  due: one.due,
  new: one.new,
  decks: one.decks.map((deck) => ({
    deck: deck.deck,
    reviewCards: deck.reviewCards,
    due: deck.due,
    new: deck.new,
    learned: deck.learned,
    unbegun: deck.unbegun,
  })),
  presets: one.presets.map((preset) => ({
    preset: preset.preset,
    title: preset.title,
    decks: preset.decks,
    cards: preset.cards,
    owed: preset.owedDue + preset.owedNew,
    answered: preset.reviewed,
    answeredNew: preset.reviewedNew,
    answeredReviews: preset.reviewedReviews,
    took: getMinutes(preset.durationMs),
    new: preset.new,
    reviews: preset.reviews,
    minutes: preset.minutes,
    closes: {
      new: preset.closesNew,
      reviews: preset.closesReviews,
      minutes: preset.closesMinutes,
    },
    stopsOn: preset.stopsOn,
  })),
  unread: one.unread,
  isIndexing: false,
})

function useVaultRows() {
  const vaults = shallowRef<readonly VaultCardsDue[]>([])

  const setVaults = (all: readonly VaultCounts[]) => {
    const target = new Map(vaults.value.map((one) => [one.vault, one]))
    vaults.value = all.map((one) => {
      const was = target.get(one.id)
      return was?.isCounted ? { ...was, name: one.name, path: one.path } : createUncountedVault(one)
    })
  }

  const setVaultCount = (one: VaultCounts) => {
    const now = one.isIndexing ? createUncountedVault(one) : createCountedVault(one)
    vaults.value = vaults.value.map((row) => (row.vault === one.id ? now : row))
  }

  return { vaults, setVaults, setVaultCount }
}

export function useReviewCounter(deps: CountsDeps) {
  const { vaults, setVaults, setVaultCount } = useVaultRows()
  const isCounting = ref(true)
  const day = ref('')

  let underway: Promise<void> | null = null
  let taking: AbortController | null = null
  // Stryker disable next-line BooleanLiteral: overwritten in ask before any read
  let sampled = false
  // Stryker disable next-line BooleanLiteral: overwritten in countUntilQuiet before any read
  let again = false

  const count = (): Promise<void> => {
    if (underway) {
      if (sampled) again = true
      return underway
    }
    underway = countUntilQuiet().finally(() => {
      underway = null
    })
    return underway
  }

  const countUntilQuiet = async (): Promise<void> => {
    for (;;) {
      again = false
      if (!(await ask())) return
      if (!again) return
    }
  }

  const stop = () => {
    taking?.abort()
  }

  const ask = async (): Promise<boolean> => {
    isCounting.value = true
    sampled = false
    const ends = new AbortController()
    taking = ends
    try {
      for await (const update of deps.cards.watchCardsDue({}, { signal: ends.signal })) {
        sampled = true
        if (update.counted) setVaultCount(update.counted)
        else {
          day.value = update.day
          setVaults(update.vaults)
        }
      }
    } catch (why) {
      if (!ends.signal.aborted) deps.reportError(why)
    } finally {
      taking = null
      isCounting.value = false
    }
    return !ends.signal.aborted
  }

  return { vaults, isCounting, day, count, stop }
}
