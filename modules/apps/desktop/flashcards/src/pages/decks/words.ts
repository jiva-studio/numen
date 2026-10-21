/** What this screen says of a preset: its goal, its day, and why it asks nothing. */
// Stryker disable StringLiteral: interface copy
import { StopReason } from '@numen/protocol'
import { formatStopReasonMessage } from '@numen/wire'
import { parseDay, daysBetween, many, percent } from '@numen/ui'

import { isSpent } from './lib/progress'
import type { Preset, Settings } from './types'

/**
 * Why a preset or a deck under it is asking nothing.
 */
export const STOPPED = {
  /** The day held none of its cards. */
  nothing: 'nothing today',
  /** The budget was spent, and elsewhere for a deck that says this. */
  full: 'the day is full',
  /** Every card face here is unbegun, and the preset begins none a day. */
  noneToBegin: 'no cards to begin',
} as const

/**
 * How much of a deck stands learned, and what is said where no share can be.
 *
 * The word stands with the figure.
 */
export const LEARNED = {
  share: (of: number) => `${percent(of)} learned`,
  /** The preset scheduling the deck could not be read, and the rule is its. */
  unruled: 'no rule to count by',
} as const

/**
 * How many cards starting a session on this preset would put in front of a person,
 * which is what its decks still owe today, or why it would put none there.
 *
 * It is the count the session itself will ask, so it is printed currently.
 */
export const getLeftWords = (one: Preset): string => {
  if (one.cards > 0) return many(one.cards, 'card')
  return isSpent(one) ? STOPPED.full : STOPPED.nothing
}

/**
 * Why a preset schedules nothing today, in the words to show, and empty while
 * it schedules something.
 *
 * The verdict is the core's: it is what the session hands its cards out by. Two
 * of the reasons name a day, and the settings carry the one a date aimed at.
 */
export const getStoppedWords = (
  why: StopReason,
  settings: Settings | null,
  today: string,
): string =>
  formatStopReasonMessage(why, {
    day: why === StopReason.PAST_DAY && settings?.byDate ? getDayWords(settings.byDate) : undefined,
    weekday: why === StopReason.NO_LOAD ? getWeekdayWords(today) : undefined,
  })

/**
 * What the goal of a preset evaluates to, in the few words a person reads at a
 * glance. A day is said as a person reads one, and not as the file writes it.
 */
export const getGoalWords = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case 'retention':
      return `${percent(settings.retention)} remembered`
    case 'date': {
      if (!settings.byDate) return formatStopReasonMessage(StopReason.NO_DAY)
      const daysLeft = daysBetween(today, settings.byDate)
      if (left <= 0) return `by ${getDayWords(settings.byDate)}`
      return `${many(daysLeft, 'day')} to ${getDayWords(settings.byDate)}`
    }
    case 'minutes':
      if (settings.minutesADay === 0) return formatStopReasonMessage(StopReason.NO_MINUTES)
      return `${many(settings.minutesADay, 'minute')} a day`
  }
}

/** A day as a person reads one, without the year they are already in. */
const short = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'long' })

const getDayWords = (day: string): string => short.format(parseDay(day))

/** The day of the week a day falls on, by its name. */
const weekday = new Intl.DateTimeFormat(undefined, { weekday: 'long' })

const getWeekdayWords = (day: string): string => weekday.format(parseDay(day))
