/** Interface copy for a preset on this screen: its goal, its day, and why it schedules nothing. */
// Stryker disable StringLiteral: interface copy
import { StopReason } from '../../../api/types.js'
import { formatStopReasonMessage } from '../../../cards/wire/stop'
import { many, percent } from '@numen/ui'
import { parseDay, daysBetween } from '../../../lib/day'

import { isSpent } from './lib/progress'
import type { Preset, Settings } from './types'

/**
 * Why a preset or a deck under it is asking nothing.
 */
export const STOPPED = {
  /** The day held none of its cards. */
  nothing: 'nothing due today',
  /** The budget is spent; a deck reporting the same shows this text. */
  full: 'daily limit reached',
  /** Every review card here is unbegun, and the preset begins none a day. */
  noneToBegin: 'no new cards',
} as const

/**
 * How much of a deck is learned, and the text used when no share can be computed.
 *
 * The word accompanies the figure.
 */
export const LEARNED = {
  share: (of: number) => `${percent(of)} learned`,
  /** The preset scheduling the deck could not be read, and the rule is its. */
  unruled: 'no learned rule',
} as const

/**
 * How many cards starting a session on this preset would present, which is what
 * its decks still have due today, or the reason it would present none.
 *
 * It is the count the session itself will ask, so it is printed as is.
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
 * The goal of a preset in a few words. A day is formatted for reading and not as
 * the file writes it.
 */
export const getGoalWords = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case 'retention':
      return `${percent(settings.retention)} remembered`
    case 'date': {
      if (!settings.byDate) return formatStopReasonMessage(StopReason.NO_DAY)
      const daysLeft = daysBetween(today, settings.byDate)
      if (daysLeft <= 0) return `by ${getDayWords(settings.byDate)}`
      return `${many(daysLeft, 'day')} to ${getDayWords(settings.byDate)}`
    }
    case 'minutes':
      if (settings.minutesADay === 0) return formatStopReasonMessage(StopReason.NO_MINUTES)
      return `${many(settings.minutesADay, 'minute')} a day`
  }
}

/** A day formatted for reading, without the current year. */
const short = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'long' })

const getDayWords = (day: string): string => short.format(parseDay(day))

/** The day of the week a day falls on, by its name. */
const weekday = new Intl.DateTimeFormat(undefined, { weekday: 'long' })

const getWeekdayWords = (day: string): string => weekday.format(parseDay(day))
