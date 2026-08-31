/**
 * Which preset schedules each deck of a vault, and what a day under each of
 * them comes to.
 *
 * Apart from the template because it is asked for when a vault is opened and
 * again whenever that vault moves, and the decks of one preset are counted
 * together however many there are.
 */
import { computed, ref } from 'vue'
import { Goal } from '@numen/protocol'
import type { Refusal } from '@numen/protocol'

import { deckName } from './core'
import type { DeckOwing } from './core'
import { said } from './reading/core'
import type { Closes, Owing } from './core'

export type { Closes }

/** A budget taking no part in the day, which nothing is weighed against. */
export const CLOSES_NOTHING: Closes = { new: '', reviews: '', minutes: '' }

/** How the decks pointing at one preset are scheduled. */
export interface Settings {
  goal: Goal
  byDate: string
  minutesADay: number
  newADay: number
  reviewsADay: number
  retention: number
  /** What each day of the week carries, in per cent, under the day's own name. */
  load: Record<string, number>
  evenLoad: boolean
}

/**
 * What one day holds under a preset, the day of the week having had its say:
 * cards of each kind, and how long the day runs.
 */
export interface Budget {
  new: number
  reviews: number
  minutes: number
}

/**
 * What the application answers about the presets of a vault. Every question
 * names the vault it is about, the same way an answer names the one it is
 * written to.
 */
export interface Asks {
  scheduling(said: { vaultId: string; deck: string }): Promise<{
    preset?:
      | {
          path: string
          title: string
          settings?: Settings | undefined
          problems: readonly string[]
        }
      | undefined
    refusal?: Refusal | undefined
  }>
}

/** One preset, the decks it schedules, and what it holds today. */
export interface Preset {
  /** The note the settings were read from, and empty for the defaults. */
  readonly path: string
  readonly name: string
  /** How its decks are scheduled, and null where no deck points at it. */
  readonly settings: Settings | null
  /** The decks it schedules, by the path each is filed under. */
  readonly decks: readonly string[]
  /** How many decks name it, whatever they hold. */
  readonly named: number
  /** How many card faces stand in those decks. */
  readonly faces: number
  /**
   * What sitting down to it would ask, as the count gives it: the cards its
   * decks owe today, already held to the budgets that close the day.
   */
  readonly cards: number
  /** What the day holds under it, which is what today is weighed against. */
  readonly budget: Budget
  /** Which key each of those budgets closes on, and empty where it closes none. */
  readonly closes: Closes
  /** Cards answered under it since the day opened, and the minutes they took. */
  readonly answered: number
  readonly took: number
  /** Why it schedules nothing, and empty while it schedules something. */
  readonly paused: string
  /**
   * What is wrong with the preset, in the words to show, and empty where
   * nothing is: what the file said that could not be read, or why the file
   * itself could not be.
   */
  readonly wrong: string
}

export interface Scheduling {
  presets: Asks
}

export function scheduling(deps: Scheduling) {
  const presets = ref<readonly Preset[]>([])

  /** Which vault the presets on hand belong to. */
  const of = ref('')

  /** The preset each deck is scheduled by, by the path the deck is filed under. */
  const byDeck = computed(() => {
    const out = new Map<string, Preset>()
    for (const one of presets.value) {
      for (const deck of one.decks) out.set(deck, one)
    }
    return out
  })

  const forget = () => {
    presets.value = []
    of.value = ''
  }

  // Today is the review day, which the application measures and this window is
  // told: it begins at the hour the settings name.
  const read = async (vault: Owing | null, today: string) => {
    if (!vault) {
      forget()
      return
    }
    of.value = vault.vaultId

    const held = await Promise.all(
      vault.decks.map((deck) => scheduled(deps.presets, vault.vaultId, deck.deck)),
    )
    if (of.value !== vault.vaultId) return

    presets.value = gather(vault, held, today)
  }

  return { presets, byDeck, of, read, forget }
}

/** What was answered about one deck's preset. */
interface Answered {
  readonly deck: string
  /** The preset, and null where it was not read. */
  readonly held: {
    path: string
    name: string
    settings: Settings
    problems: readonly string[]
  } | null
  /** Why it was not read, in the words to show, and empty where it was. */
  readonly refused: string
}

/** What is shown of a preset the window has no other reason to give for. */
const UNREAD = 'the settings of this preset could not be read'

/** The preset one deck is scheduled by, or why it could not be read. */
const scheduled = async (presets: Asks, vaultId: string, deck: string): Promise<Answered> => {
  try {
    const answer = await presets.scheduling({ vaultId, deck })
    const settings = answer.preset?.settings
    if (!answer.preset || !settings) {
      return { deck, held: null, refused: said(answer.refusal) || UNREAD }
    }
    return {
      deck,
      held: {
        path: answer.preset.path,
        name: answer.preset.title,
        settings,
        problems: answer.preset.problems,
      },
      refused: '',
    }
  } catch {
    return { deck, held: null, refused: UNREAD }
  }
}

/** One preset while its decks are still being counted into it. */
interface Gathering {
  path: string
  name: string
  settings: Settings
  decks: string[]
  due: number
  fresh: number
  /** What was wrong in the file, each said once however many decks name it. */
  problems: Set<string>
}

/** What the day holds under a preset the application counted nothing for. */
const budgetOf = (settings: Settings): Budget => ({
  new: settings.newADay,
  reviews: settings.reviewsADay,
  minutes: settings.minutesADay,
})

/**
 * The presets of a vault, each carrying the decks that name it.
 *
 * Every preset the vault holds gets a row. The ones whose decks hold cards are
 * gathered from what each deck answered, and the rest stand on the count alone.
 */
const gather = (vault: Owing, answered: readonly Answered[], today: string): Preset[] => {
  const owed = new Map(vault.decks.map((one) => [one.deck, one]))
  const came = new Map(vault.presets.map((one) => [one.preset, one]))
  const at = new Map<string, Gathering>()

  for (const one of answered) {
    if (!one.held) continue
    let into = at.get(one.held.path)
    if (!into) {
      into = {
        path: one.held.path,
        name: one.held.name || 'The defaults',
        settings: one.held.settings,
        decks: [],
        due: 0,
        fresh: 0,
        problems: new Set(),
      }
      at.set(one.held.path, into)
    }
    into.decks.push(one.deck)
    for (const problem of one.held.problems) into.problems.add(problem)
    const deck = owed.get(one.deck)
    into.due += deck?.due ?? 0
    into.fresh += deck?.new ?? 0
  }

  const out = [...at.values()].map((one): Preset => {
    const why = paused(one.settings, today)
    const day = came.get(one.path)
    const budget = day ?? budgetOf(one.settings)
    return {
      path: one.path,
      name: one.name,
      settings: one.settings,
      decks: one.decks,
      named: day?.decks ?? one.decks.length,
      faces: day?.cards ?? 0,
      // What a sitting over it asks is the count's own figure, and a build that
      // answered none falls back to what its decks owe between them.
      cards: day?.owed ?? one.due + one.fresh,
      budget: { new: budget.new, reviews: budget.reviews, minutes: budget.minutes },
      closes: day?.closes ?? CLOSES_NOTHING,
      answered: day?.answered ?? 0,
      took: day?.took ?? 0,
      paused: why,
      wrong: [...one.problems].join('; '),
    }
  })

  // A preset every one of whose decks is empty is answered for no deck, and so
  // is one whose settings could not be read, so both stand here on the count
  // alone, beside the presets nothing points at.
  const why = refusedFor(answered)
  for (const one of vault.presets) {
    if (at.has(one.preset)) continue
    out.push({
      path: one.preset,
      name: one.title || deckName(one.preset),
      settings: null,
      decks: [],
      named: one.decks,
      faces: one.cards,
      // The count answered for this preset with its own figures, and the day is
      // drawn from them.
      cards: one.owed,
      budget: { new: one.new, reviews: one.reviews, minutes: one.minutes },
      closes: one.closes,
      answered: one.answered,
      took: one.took,
      paused: '',
      wrong: one.decks > 0 ? why : '',
    })
  }
  return out
}

/**
 * Why the presets no deck answered for could not be read. Every deck of one
 * preset is refused for the same reason, so a reason every refused deck gave is
 * the reason of each preset none of them could read.
 */
const refusedFor = (answered: readonly Answered[]): string => {
  const why = new Set(answered.filter((one) => one.refused).map((one) => one.refused))
  if (why.size === 0) return ''
  return why.size === 1 ? ([...why][0] ?? '') : UNREAD
}

/**
 * Whether sitting down to one deck is offered: it owes something today, and the
 * preset scheduling it schedules something.
 *
 * The row and the letter drawn on it are one act, so both ask this.
 */
export const opens = (deck: DeckOwing, by: ReadonlyMap<string, Preset>): boolean =>
  deck.due + deck.new > 0 && !by.get(deck.deck)?.paused

/** How many cards the day holds at most. */
export const holds = (budget: Budget): number => budget.new + budget.reviews

/**
 * How far through its day a preset stands: what has been answered against the
 * cards the day holds, and what it has taken against the minutes the day runs,
 * whichever of the two is further along.
 *
 * A day answered past what its budget holds stands above one, which is a day
 * over its budget and not a day that is done.
 */
export const through = (one: Preset): number => {
  // Only a budget that closes the day is weighed against. A preset steered by
  // its minutes keeps its card counts as the person left them, and a share
  // worked out from those would be a share of a number that binds nothing.
  const ofCards =
    (one.closes.new ? one.budget.new : 0) + (one.closes.reviews ? one.budget.reviews : 0)
  const ofMinutes = one.closes.minutes ? one.budget.minutes : 0
  return Math.max(
    0,
    ofCards > 0 ? one.answered / ofCards : 0,
    ofMinutes > 0 ? one.took / ofMinutes : 0,
  )
}

/**
 * Whether a preset's day is spent, which is what leaves every deck under it
 * nothing more to ask however much those decks still hold.
 */
export const spent = (one: Preset): boolean => through(one) >= 1

/**
 * How many cards sitting down to this preset would put in front of a person,
 * which is what its decks still owe today.
 *
 * It is the count the sitting itself will ask, so it is printed as it stands.
 */
export const leftWords = (one: Preset): string => {
  if (one.cards <= 0) return ''
  return many(one.cards, 'card')
}

/**
 * Why a preset schedules nothing, and empty while it schedules something.
 *
 * The budget the goal names is the one that pauses it, and a budget the goal
 * does not name stands as the person left it and pauses nothing. A day of the
 * week carrying none of the load is a pause of that one day.
 */
export const paused = (settings: Settings, today: string): string => {
  const why = budgeted(settings, today)
  if (why) return why
  return shareOn(settings, today) === 0 ? `no load on ${weekdayWords(today)}` : ''
}

/** Why the budget the goal names schedules nothing, and empty while it does. */
const budgeted = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case Goal.RETENTION:
      return settings.newADay === 0 && settings.reviewsADay === 0 ? 'no cards a day' : ''
    case Goal.BY_DATE:
      if (settings.byDate && settings.byDate < today) {
        return `${dayWords(settings.byDate)} has passed`
      }
      return ''
    default:
      return settings.minutesADay === 0 ? 'no budget in time' : ''
  }
}

/** The first three letters of each day's name, which is how a preset writes one. */
const DAYS = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']

/**
 * How much of a day's load the day of the week carries, as a share of one. A
 * day the preset does not name carries the whole of it.
 */
const shareOn = (settings: Settings, today: string): number => {
  const carried = settings.load[DAYS[dated(today).getDay()] ?? '']
  return carried === undefined ? 1 : carried / 100
}

/**
 * What the goal of a preset comes to, in the few words a person reads at a
 * glance. Each goal is said here as the preset tab says it, so one goal reads
 * the same in both windows.
 */
export const goalWords = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case Goal.RETENTION:
      return `${Math.round(settings.retention * 100)}% remembered`
    case Goal.BY_DATE: {
      if (!settings.byDate) return 'by no day'
      const left = daysBetween(today, settings.byDate)
      if (left <= 0) return `by ${dayWords(settings.byDate)}`
      return `${many(left, 'day')} to ${dayWords(settings.byDate)}`
    }
    default:
      if (settings.minutesADay === 0) return 'no budget in time'
      return `${many(settings.minutesADay, 'minute')} a day`
  }
}

/** A count and the thing it counts, in the singular where there is one of it. */
const many = (value: number, one: string): string =>
  `${value} ${value === 1 ? one : `${one}s`}`

/** A day as the application writes one: the year, the month and the day. */
export const named = (at: Date): string => {
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}`
}

/** A day as a person reads one, without the year they are already in. */
const short = new Intl.DateTimeFormat(undefined, { day: 'numeric', month: 'long' })

const dayWords = (day: string): string => short.format(dated(day))

/** The day of the week a day falls on, by its name. */
const weekday = new Intl.DateTimeFormat(undefined, { weekday: 'long' })

const weekdayWords = (day: string): string => weekday.format(dated(day))

/** How many days lie between two days. */
const daysBetween = (from: string, to: string): number =>
  Math.round((dated(to).getTime() - dated(from).getTime()) / 86400000)

const dated = (day: string): Date => {
  const [year, month, at] = day.split('-').map(Number)
  return new Date(year ?? 2000, (month ?? 1) - 1, at ?? 1)
}
