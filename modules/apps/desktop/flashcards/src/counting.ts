/**
 * What every vault the installation holds comes to today.
 *
 * Apart from the template because counting is asked for from three places — the
 * window opening, a sitting ending, and a vault moving underneath it — and one
 * count runs at a time however many ask.
 */
import { ref } from 'vue'

import type { Owing } from './core'

/** What the front door of the application answers. */
export interface Counts {
  owing(said: Record<string, never>): Promise<Counted>
}

/** What it answers with. */
export interface Counted {
  vaults: readonly {
    vaultId: string
    name: string
    path: string
    faces: number
    due: number
    new: number
    decks: readonly { deck: string; faces: number; due: number; new: number }[]
    presets: readonly {
      preset: string
      title: string
      decks: number
      cards: number
      answered: number
      tookMs: bigint
      new: number
      reviews: number
      minutes: number
    }[]
    unread: string
  }[]
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

  /** The count on its way, so two never run at once and none is asked twice. */
  let underway: Promise<void> | null = null

  const count = (): Promise<void> => {
    if (!underway) {
      underway = ask().finally(() => {
        underway = null
      })
    }
    return underway
  }

  const ask = async () => {
    counting.value = true
    try {
      const answer = await deps.cards.owing({})
      vaults.value = answer.vaults.map((one) => ({
        vaultId: one.vaultId,
        name: one.name,
        path: one.path,
        faces: one.faces,
        due: one.due,
        new: one.new,
        decks: one.decks.map((deck) => ({
          deck: deck.deck,
          faces: deck.faces,
          due: deck.due,
          new: deck.new,
        })),
        presets: one.presets.map((preset) => ({
          preset: preset.preset,
          title: preset.title,
          decks: preset.decks,
          cards: preset.cards,
          answered: preset.answered,
          took: minutes(preset.tookMs),
          new: preset.new,
          reviews: preset.reviews,
          minutes: preset.minutes,
        })),
        unread: one.unread,
      }))
    } catch (why) {
      deps.failed(why)
    } finally {
      counting.value = false
    }
  }

  return { vaults, counting, count }
}
