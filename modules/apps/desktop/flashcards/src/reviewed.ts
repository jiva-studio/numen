/**
 * What a vault was answered on, for the grid of days the vault screen draws.
 *
 * Apart from the template because it is asked for when a vault is opened and
 * again whenever that vault moves, and what is held between the two is a rule.
 */
import { ref } from 'vue'
import type { HeatmapTally } from '@numen/ui'

/** What the application answers about a vault's days. */
export interface Asks {
  reviewed(said: { vaultId: string }): Promise<Said>
}

export interface Said {
  days: readonly Day[]
  due: readonly { day: string; answered: number }[]
  streak: number
  answered: number
}

/** One day, and what was answered on it. */
export interface Day {
  day: string
  answered: number
  again: number
  hard: number
  good: number
  easy: number
  /** The answers given to cards already being reviewed, and how many came back. */
  asked: number
  recalled: number
}

export interface Reviewing {
  cards: Asks
  failed(why: unknown): void
}

export function reviewed(deps: Reviewing) {
  /** How much was answered on each day, by the day it was answered on. */
  const days = ref<ReadonlyMap<string, HeatmapTally>>(new Map())
  /** How much falls on each day still to come, by the day it falls on. */
  const due = ref<ReadonlyMap<string, number>>(new Map())
  const streak = ref(0)
  const answered = ref(0)

  /** Which vault the days on hand belong to, so another vault's are not drawn. */
  const of = ref('')

  const forget = () => {
    days.value = new Map()
    due.value = new Map()
    streak.value = 0
    answered.value = 0
    of.value = ''
  }

  const read = async (vaultId: string) => {
    if (!vaultId) {
      forget()
      return
    }
    of.value = vaultId
    try {
      const said = await deps.cards.reviewed({ vaultId })
      // A person who moved to another vault while this was on its way is
      // looking at that one, and these days are not its days.
      if (of.value !== vaultId) return
      days.value = new Map(said.days.map((one) => [one.day, counted(one)]))
      due.value = new Map(said.due.map((one) => [one.day, one.answered]))
      streak.value = said.streak
      answered.value = said.answered
    } catch (why) {
      if (of.value !== vaultId) return
      deps.failed(why)
      forget()
    }
  }

  return { days, due, streak, answered, of, read, forget }
}

/** One day as the grid holds it. */
const counted = (one: Day): HeatmapTally => ({
  answered: one.answered,
  again: one.again,
  hard: one.hard,
  good: one.good,
  easy: one.easy,
  asked: one.asked,
  recalled: one.recalled,
})
