/**
 * The answers recorded for a vault, for the grid of days on the vault screen.
 *
 * Kept outside the template because it loads when a vault is opened and again
 * when that vault changes, and the state retained between loads follows a
 * rule.
 */
import { getCurrentDate } from '@numen/ui'
import { ref } from 'vue'
import { getStretch as getHeatmapStretch } from '../../../../ui/heatmap'
import type { Tally as HeatmapTally } from '../../../../ui/heatmap'

/** What the application answers about a vault's days. */
export interface ReviewDaysClient {
  listReviewDays(request: { vault: string; from: string; to: string }): Promise<ReviewDays>
}

export interface ReviewDays {
  days: readonly Day[]
  due: readonly { day: string; reviewed: number }[]
  streak: number
  reviewed: number
}

/** What the grid draws a day at, which is what Heatmap is left to by default. */
const CELL = 11
const GAP = 3

/** One day, and what was answered on it. */
interface Day {
  day: string
  reviewed: number
  again: number
  hard: number
  good: number
  easy: number
  /** The answers given to cards already being reviewed, and how many came back. */
  tested: number
  recalled: number
}

export interface ReviewDaysDeps {
  cards: ReviewDaysClient
  reportError(why: unknown): void
  /**
   * The widest the grid of days can be rendered, in pixels. The requested days
   * are the days a grid of that width displays, and a vault has an entry for
   * every day on which it has been reviewed.
   */
  widest(): number
  /** What day it is. The machine's own, where a caller names none. */
  now?(): Date
}

export function useReviewDays(deps: ReviewDaysDeps) {
  const now = deps.now ?? (() => getCurrentDate())
  /** How much was answered on each day, by the day it was answered on. */
  const days = ref<ReadonlyMap<string, HeatmapTally>>(new Map())
  /** How much falls on each day still to come, by the day it falls on. */
  const due = ref<ReadonlyMap<string, number>>(new Map())
  const streak = ref(0)
  const reply = ref(0)

  /** The vault the loaded days belong to; the days of another vault are not displayed. */
  const of = ref('')

  const forget = () => {
    days.value = new Map()
    due.value = new Map()
    streak.value = 0
    reply.value = 0
    of.value = ''
  }

  const readItem = async (vault: string) => {
    if (!vault) {
      forget()
      return
    }
    of.value = vault
    try {
      const { from, to } = getHeatmapStretch({ width: deps.widest(), cell: CELL, gap: GAP }, now())
      const reviewData = await deps.cards.listReviewDays({ vault, from, to })
      // If the user switched to another vault while this request was pending, the
      // result belongs to the previous vault and is discarded.
      if (of.value !== vault) return
      days.value = new Map(reviewData.days.map((one) => [one.day, one]))
      due.value = new Map(reviewData.due.map((one) => [one.day, one.reviewed]))
      streak.value = reviewData.streak
      reply.value = reviewData.reviewed
    } catch (why) {
      if (of.value !== vault) return
      deps.reportError(why)
      forget()
    }
  }

  return { days, due, streak, answerCount: reply, reviewed: reply, of, read: readItem, forget }
}
