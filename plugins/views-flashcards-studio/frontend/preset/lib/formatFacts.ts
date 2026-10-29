// Stryker disable StringLiteral: interface copy
import { many, percent, plural } from '@numen/ui'
import type { BuyFact, LearningFact, MaterialFact } from './facts'

/** A count formatted for display. */
export const count = (value: number): string => `${Math.round(value)}`

/** An error clause as a sentence. */
export const sayAsSentence = (clause: string): string =>
  `${clause.charAt(0).toUpperCase()}${clause.slice(1)}.`

/** A fact about what a setting provides, formatted as text. */
export const formatBuyFact = (fact: BuyFact): string => {
  switch (fact.kind) {
    case 'daysOff':
      return `${many(fact.days, 'day')} off`
    case 'minutesADay':
      return `${many(fact.minutes, 'minute')} a day`
    case 'remembered':
      return `${percent(fact.share)} remembered`
    case 'cardsASession':
      return `${many(fact.cards, 'card')} a session`
    case 'cannotGetThere':
      return `${count(fact.short)} of ${count(fact.cards)} not reachable`
    case 'overdueGone':
      return `overdue gone in ${many(fact.days, 'day')}`
    case 'overdueStays':
      return `not within ${many(fact.horizon, 'day')}`
  }
}

/** A fact about the material, formatted as a figure and its name. */
export const formatMaterialFact = (fact: MaterialFact): { figure: string; name: string } => {
  const figure = count(fact.count)
  switch (fact.kind) {
    case 'decks':
      return { figure, name: plural(fact.count, 'deck') }
    case 'cards':
      return { figure, name: plural(fact.count, 'card') }
    case 'overdue':
      return { figure, name: 'overdue' }
    case 'unbegun':
      return { figure, name: 'new' }
  }
}

/** A fact about what is learned, formatted as a figure and its name. */
export const formatLearningFact = (fact: LearningFact): { figure: string; name: string } => {
  switch (fact.kind) {
    case 'notYet':
      return { figure: 'not yet', name: 'in the days ahead' }
    case 'allToday':
      return { figure: 'today', name: 'all learned' }
    case 'daysToLearn':
      return {
        figure: count(fact.days),
        name: plural(fact.days, 'day to learn it', 'days to learn it'),
      }
    case 'learnedToday':
      return { figure: `${count(fact.learned)} of ${count(fact.cards)}`, name: 'learned today' }
  }
}
