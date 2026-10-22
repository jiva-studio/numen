/**
 * The settings of a preset in the windows' own words, and the message they are
 * read from and written to. Both windows map the message through here.
 */
import { BudgetUnit as BudgetUnits, Goal as Goals, Rule as Rules } from '@numen/protocol'

import { goalNames, goalOf } from './goal'
import type { Goal } from './goal'
import { getNames } from './enumNames'

/** The unit a day's budget is spent in. */
export type BudgetUnit = 'cards' | 'shows'

/** What a preset counts as learned. */
export type Rule = 'interval' | 'retention'

/** A share of a day's load for each day of the week that is not at the whole. */
export type Load = Readonly<Record<string, number>>

/** How the decks pointing at one preset are scheduled. */
export interface Settings {
  readonly goal: Goal
  /** The day the material is to be in the head, as `2026-09-30`. Empty for none. */
  readonly byDate: string
  readonly minutesADay: number
  readonly newADay: number
  readonly reviewsADay: number
  readonly retention: number
  readonly counts: BudgetUnit
  readonly backlog: number
  readonly load: Load
  readonly hasEvenLoad: boolean
  readonly learned: Rule
  readonly interval: number
}

/** The fields of the settings message the mapping reads and writes. */
export interface SettingsMessage {
  readonly goal: Goals
  readonly byDate: string
  readonly minutesADay: number
  readonly newADay: number
  readonly reviewsADay: number
  readonly retention: number
  readonly counts: BudgetUnits
  readonly backlog: number
  readonly load: Readonly<Record<string, number>>
  readonly hasEvenLoad: boolean
  readonly learned: Rules
  readonly interval: number
}

/** A preset naming nothing, and how a deck pointing at none is scheduled. */
export const DEFAULT_SETTINGS: Settings = {
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  counts: 'cards',
  backlog: 100,
  load: {},
  hasEvenLoad: true,
  learned: 'interval',
  interval: 21,
}

/** The rule in the windows' words, and nothing for a message naming none. */
export const ruleOf: Readonly<Record<Rules, Rule | null>> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

export const ruleNames: Readonly<Record<Rule, Rules>> = getNames<Rule, Rules>(ruleOf)

/** The unit in the windows' words, and nothing for a message naming none. */
export const budgetUnitOf: Readonly<Record<BudgetUnits, BudgetUnit | null>> = {
  [BudgetUnits.UNSPECIFIED]: null,
  [BudgetUnits.CARDS]: 'cards',
  [BudgetUnits.SHOWS]: 'shows',
}

export const budgetUnitNames: Readonly<Record<BudgetUnit, BudgetUnits>> = getNames<
  BudgetUnit,
  BudgetUnits
>(budgetUnitOf)

/** The settings of a message; a message naming none, or a value naming none, takes the default. */
export const parseSettings = (message: SettingsMessage | undefined): Settings =>
  message === undefined
    ? DEFAULT_SETTINGS
    : {
        goal: goalOf[message.goal] ?? DEFAULT_SETTINGS.goal,
        byDate: message.byDate,
        minutesADay: message.minutesADay,
        newADay: message.newADay,
        reviewsADay: message.reviewsADay,
        retention: message.retention,
        counts: budgetUnitOf[message.counts] ?? DEFAULT_SETTINGS.counts,
        backlog: message.backlog,
        load: message.load,
        hasEvenLoad: message.hasEvenLoad,
        learned: ruleOf[message.learned] ?? DEFAULT_SETTINGS.learned,
        interval: message.interval,
      }

/** The settings as the message carries them. */
export const formatSettings = (settings: Settings): SettingsMessage => ({
  goal: goalNames[settings.goal],
  byDate: settings.byDate,
  minutesADay: settings.minutesADay,
  newADay: settings.newADay,
  reviewsADay: settings.reviewsADay,
  retention: settings.retention,
  counts: budgetUnitNames[settings.counts],
  backlog: settings.backlog,
  load: { ...settings.load },
  hasEvenLoad: settings.hasEvenLoad,
  learned: ruleNames[settings.learned],
  interval: settings.interval,
})
