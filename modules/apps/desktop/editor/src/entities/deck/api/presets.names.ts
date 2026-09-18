/** What the schema's values for a preset are called in the window's own words. */
import { BudgetUnit as BudgetUnits, Rule as Rules } from '@numen/protocol'
import { namesOf } from '@numen/wire'
import type { BudgetUnit, Rule } from '../lib/presets'

/** What counts as learned. */
export const LEARNED: Record<Rules, Rule | null> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

export const RULING = namesOf<Rule, Rules>(LEARNED)

/** The unit a budget is spent in. */
export const COUNTED: Record<BudgetUnits, BudgetUnit | null> = {
  [BudgetUnits.UNSPECIFIED]: null,
  [BudgetUnits.CARDS]: 'cards',
  [BudgetUnits.SHOWS]: 'shows',
}

export const COUNTING = namesOf<BudgetUnit, BudgetUnits>(COUNTED)
