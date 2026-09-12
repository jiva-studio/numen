/** What this screen says of a preset: its goal, its day, and why it asks nothing. */
import { StopReason } from '@numen/protocol'
import { dayOf, daysBetween, many, percent } from '@numen/ui'

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
  noCards: 'no cards a day',
  noMinutes: 'no budget in time',
  noDay: 'by no day',
  /** The day it aimed at is behind us, named where the window holds it. */
  pastDay: 'the day has passed',
  passed: (day: string) => `${getDayWords(day)} has passed`,
  noLoad: (day: string) => `no load on ${getWeekdayWords(day)}`,
  /** No day of the week carries any of the load, so there is no next day. */
  noWeek: 'no load on any day',
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
 * It is the count the session itself will ask, so it is printed as it stands.
 */
export const getLeftWords = (one: Preset): string => {
  if (one.cards > 0) return many(one.cards, 'card')
  return isSpent(one) ? STOPPED.full : STOPPED.nothing
}

/**
 * The words each verdict the vault may hand over comes to. A verdict scheduling
 * something says nothing. Every value stands here, so a verdict added to the
 * schema is one this window is made to answer.
 */
const WHY: Record<StopReason, (settings: Settings | null, today: string) => string> = {
  [StopReason.UNSPECIFIED]: () => '',
  [StopReason.NOTHING]: () => '',
  [StopReason.NO_MINUTES]: () => STOPPED.noMinutes,
  [StopReason.NO_CARDS]: () => STOPPED.noCards,
  [StopReason.NO_DAY]: () => STOPPED.noDay,
  [StopReason.PAST_DAY]: (settings) =>
    settings?.byDate ? STOPPED.passed(settings.byDate) : STOPPED.pastDay,
  [StopReason.NO_LOAD]: (settings, today) => STOPPED.noLoad(today),
  [StopReason.NO_WEEK]: () => STOPPED.noWeek,
}

/**
 * Why a preset schedules nothing today, in the words to show, and empty while
 * it schedules something.
 *
 * The verdict is the core's: it is what the session hands its cards out by. Two
 * of the reasons name a day, and the settings carry the one a date aimed at.
 */
export const getStoppedWords = (why: StopReason, settings: Settings | null, today: string): string =>
  WHY[why](settings, today)

/**
 * What the goal of a preset comes to, in the few words a person reads at a
 * glance. A day is said as a person reads one, and not as the file writes it.
 */
export const getGoalWords = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case 'retention':
      return `${percent(settings.retention)} remembered`
    case 'date': {
      if (!settings.byDate) return 'by no day'
      const left = daysBetween(today, settings.byDate)
      if (left <= 0) return `by ${getDayWords(settings.byDate)}`
      return `${many(left, 'day')} to ${getDayWords(settings.byDate)}`
    }
    case 'minutes':
      if (settings.minutesADay === 0) return STOPPED.noMinutes
      return `${many(settings.minutesADay, 'minute')} a day`
  }
}

/** A day as a person reads one, without the year they are already in. */
const short = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'long' })

const getDayWords = (day: string): string => short.format(dayOf(day))

/** The day of the week a day falls on, by its name. */
const weekday = new Intl.DateTimeFormat(undefined, { weekday: 'long' })

const getWeekdayWords = (day: string): string => weekday.format(dayOf(day))
