/**
 * What every vault the installation holds comes to today.
 *
 * Apart from the template because counting is asked for from three places — the
 * window opening, a session ending, and a vault moving underneath it — and one
 * count runs at a time however many ask.
 *
 * The vaults arrive first, by name and by where they are, and each count
 * follows on its own. A vault whose count has not arrived stands on the list
 * with nothing said about what it holds.
 */
import { ref, shallowRef } from 'vue'
import type { StopReason } from '@numen/protocol'

import type { VaultCardsDue } from '../types'

/** What the front door of the application answers. */
export interface CardsDueClient {
  watchCardsDue(
    said: Record<string, never>,
    how?: { signal?: AbortSignal },
  ): AsyncIterable<DueCounts>
}

/** One vault, as the count answers about it. */
export interface VaultCounts {
  id: string
  name: string
  path: string
  faces: number
  due: number
  new: number
  decks: readonly {
    deck: string
    faces: number
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
    answered: number
    answeredNew: number
    answeredReviews: number
    tookMs: bigint
    new: number
    reviews: number
    minutes: number
    closesNew: string
    closesReviews: string
    closesMinutes: string
    stopsOn: StopReason
  }[]
  unread: string
  isReading: boolean
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
const getMinutes = (ms: bigint): number => Number(ms) / 60000

export interface CountsDeps {
  cards: CardsDueClient
  reportError(why: unknown): void
}

export function useReviewCounter(deps: CountsDeps) {
  const vaults = shallowRef<readonly VaultCardsDue[]>([])
  const isCounting = ref(true)

  /**
   * The day the counts stand in, and the day a goal is weighed against. A day of
   * review begins at the hour the settings name, so an hour past midnight is
   * still the day before.
   */
  const day = ref('')

  /** The count on its way, so two never run at once and none is asked twice. */
  let underway: Promise<void> | null = null

  /** What ends the count on its way. */
  let taking: AbortController | null = null

  /** Whether the count on its way has begun handing back what it worked out. */
  let sampled = false

  /** Whether it was asked for again after it had. */
  let again = false

  const count = (): Promise<void> => {
    if (underway) {
      // Asked for before anything was worked out, this count answers the asking
      // too. Asked for after, it has already passed the row that moved.
      if (sampled) again = true
      return underway
    }
    underway = countUntilQuiet().finally(() => {
      underway = null
    })
    return underway
  }

  /**
   * The counting, until nothing has asked for it again. A vault read while a
   * count was running is one whose numbers landed after that count had worked
   * its row out, so the asking is answered rather than dropped.
   */
  const countUntilQuiet = async (): Promise<void> => {
    for (;;) {
      again = false
      if (!(await ask())) return
      if (!again) return
    }
  }

  /**
   * The counting ended where it stands: the window is going, or the person has
   * chosen the vault they came for. What has not arrived is not waited for.
   */
  const stop = () => {
    taking?.abort()
  }

  /** A vault on the list before its count has arrived. */
  const createUncountedVault = (one: VaultCounts): VaultCardsDue => ({
    vault: one.id,
    name: one.name,
    path: one.path,
    isCounted: false,
    faces: 0,
    due: 0,
    new: 0,
    decks: [],
    presets: [],
    unread: one.unread,
    isReading: one.isReading,
  })

  /** A vault as its own count leaves it. */
  const createCountedVault = (one: VaultCounts): VaultCardsDue => ({
    vault: one.id,
    name: one.name,
    path: one.path,
    isCounted: true,
    faces: one.faces,
    due: one.due,
    new: one.new,
    decks: one.decks.map((deck) => ({
      deck: deck.deck,
      faces: deck.faces,
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
      answered: preset.answered,
      answeredNew: preset.answeredNew,
      answeredReviews: preset.answeredReviews,
      took: getMinutes(preset.tookMs),
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
    isReading: false,
  })

  /**
   * The vaults, over what the window already shows. A vault counted a moment
   * ago keeps that count until its new one lands, so a list already drawn is
   * never emptied to be filled again.
   */
  const setVaults = (all: readonly VaultCounts[]) => {
    const held = new Map(vaults.value.map((one) => [one.vault, one]))
    vaults.value = all.map((one) => {
      const was = held.get(one.id)
      return was?.isCounted ? { ...was, name: one.name, path: one.path } : createUncountedVault(one)
    })
  }

  /**
   * One vault's count, into the row it belongs to. A vault being read into the
   * index has no count yet, and its row goes on waiting for one.
   */
  const setVaultCount = (one: VaultCounts) => {
    const now = one.isReading ? createUncountedVault(one) : createCountedVault(one)
    vaults.value = vaults.value.map((row) => (row.vault === one.id ? now : row))
  }

  /** One count, answering whether it ran to the end. */
  const ask = async (): Promise<boolean> => {
    isCounting.value = true
    sampled = false
    const ends = new AbortController()
    taking = ends
    try {
      for await (const said of deps.cards.watchCardsDue({}, { signal: ends.signal })) {
        sampled = true
        if (said.counted) setVaultCount(said.counted)
        else {
          day.value = said.day
          setVaults(said.vaults)
        }
      }
    } catch (why) {
      // A count the window ended is not something to tell a person about.
      if (!ends.signal.aborted) deps.reportError(why)
    } finally {
      if (taking === ends) taking = null
      isCounting.value = false
    }
    return !ends.signal.aborted
  }

  return { vaults, isCounting, day, count, stop }
}
