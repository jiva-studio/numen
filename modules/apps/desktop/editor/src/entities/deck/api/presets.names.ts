/** What the schema's values for a preset are called in the window's own words. */
import { BudgetUnit as BudgetUnits, Rule as Rules } from '@numen/protocol'
import { getNames } from '@numen/wire'
import type { BudgetUnit, Rule } from '../lib/presets'

/** What counts as learned. */
export const RULE_BY_WIRE: Record<Rules, Rule | null> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

export const WIRE_BY_RULE = getNames<Rule, Rules>(RULE_BY_WIRE)

/** The unit a budget is spent in. */
export const BUDGET_UNIT_BY_WIRE: Record<BudgetUnits, BudgetUnit | null> = {
  [BudgetUnits.UNSPECIFIED]: null,
  [BudgetUnits.CARDS]: 'cards',
  [BudgetUnits.SHOWS]: 'shows',
}

export const WIRE_BY_BUDGET_UNIT = getNames<BudgetUnit, BudgetUnits>(BUDGET_UNIT_BY_WIRE)
