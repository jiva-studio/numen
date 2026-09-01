/**
 * What every vault the installation holds comes to today.
 *
 * Apart from the template because counting is asked for from three places — the
 * window opening, a sitting ending, and a vault moving underneath it — and one
 * count runs at a time however many ask.
 *
 * The vaults arrive first, by name and by where they are, and each count
 * follows on its own. A vault whose count has not arrived stands on the list
 * with nothing said about what it holds.
 */
import { ref } from 'vue'
import type { Stopped } from '@numen/protocol'

import type { Owing } from './core'

/** What the front door of the application answers. */
export interface Counts {
  owing(said: Record<string, never>, how?: { signal?: AbortSignal }): AsyncIterable<Counted>
}

/** One vault, as the count answers about it. */
export interface Vaulted {
  vaultId: string
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
  }[]
  presets: readonly {
    preset: string
    title: string
    decks: number
    cards: number
    owedDue: number
    owedNew: number
    answered: number
    tookMs: bigint
    new: number
    reviews: number
    minutes: number
    closesNew: string
    closesReviews: string
    closesMinutes: string
    stopsOn: Stopped
  }[]
  unread: string
}

/** One message of the count. */
export interface Counted {
  /** The review day these counts stand in, which begins at the hour the settings name. */
  day: string
  /** Every vault the installation holds, in the first message and in no other. */
  vaults: readonly Vaulted[]
  /** One vault worked out, in every message after the first. */
  counted?: Vaulted | undefined
}

/** A length of time as the application holds one, which is in minutes. */
const minutes = (ms: bigint): number => Number(ms) / 60000

export interface Counting {
  cards: Counts
  failed(why: unknown): void
}

export function counting(deps: Counting) {
  const vaults = ref<readonly Owing[]>([])
  const counting = ref(true)

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

  const count = (): Promise<void> => {
    if (!underway) {
      underway = ask().finally(() => {
        underway = null
      })
    }
    return underway
  }

  /**
   * The counting ended where it stands: the window is going, or the person has
   * chosen the vault they came for. What has not arrived is not waited for.
   */
  const stop = () => {
    taking?.abort()
  }

  /** A vault on the list before its count has arrived. */
  const listed = (one: Vaulted): Owing => ({
    vaultId: one.vaultId,
    name: one.name,
    path: one.path,
    counted: false,
    faces: 0,
    due: 0,
    new: 0,
    decks: [],
    presets: [],
    unread: '',
  })

  /** A vault as its own count leaves it. */
  const owed = (one: Vaulted): Owing => ({
    vaultId: one.vaultId,
    name: one.name,
    path: one.path,
    counted: true,
    faces: one.faces,
    due: one.due,
    new: one.new,
    decks: one.decks.map((deck) => ({
      deck: deck.deck,
      faces: deck.faces,
      due: deck.due,
      new: deck.new,
      learned: deck.learned,
    })),
    presets: one.presets.map((preset) => ({
      preset: preset.preset,
      title: preset.title,
      decks: preset.decks,
      cards: preset.cards,
      owed: preset.owedDue + preset.owedNew,
      answered: preset.answered,
      took: minutes(preset.tookMs),
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
  })

  /**
   * The vaults, over what the window already shows. A vault counted a moment
   * ago keeps that count until its new one lands, so a list already drawn is
   * never emptied to be filled again.
   */
  const stands = (all: readonly Vaulted[]) => {
    const held = new Map(vaults.value.map((one) => [one.vaultId, one]))
    vaults.value = all.map((one) => {
      const was = held.get(one.vaultId)
      return was?.counted ? { ...was, name: one.name, path: one.path } : listed(one)
    })
  }

  /** One vault's count, into the row it belongs to. */
  const fills = (one: Vaulted) => {
    vaults.value = vaults.value.map((row) => (row.vaultId === one.vaultId ? owed(one) : row))
  }

  const ask = async () => {
    counting.value = true
    const ends = new AbortController()
    taking = ends
    try {
      for await (const said of deps.cards.owing({}, { signal: ends.signal })) {
        if (said.counted) fills(said.counted)
        else {
          day.value = said.day
          stands(said.vaults)
        }
      }
    } catch (why) {
      // A count the window ended is not something to tell a person about.
      if (!ends.signal.aborted) deps.failed(why)
    } finally {
      if (taking === ends) taking = null
      counting.value = false
    }
  }

  return { vaults, counting, day, count, stop }
}
