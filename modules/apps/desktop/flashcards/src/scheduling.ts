/**
 * Which preset schedules each deck of a vault, and what a day under each of
 * them comes to.
 *
 * It is asked for when a vault is opened and again whenever that vault moves,
 * and the decks of one preset are counted together however many there are.
 */
import { computed, ref } from 'vue'
import { Stopped } from '@numen/protocol'
import type { Goal as Goals, Refusal } from '@numen/protocol'

import { deckName, goaled } from './core'
import type { DeckOwing, Goal } from './core'
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

/** The same, as the schema carries them. */
export interface SettingsMessage extends Omit<Settings, 'goal'> {
  goal: Goals
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
          settings?: SettingsMessage | undefined
          problems: readonly string[]
          /** Why it schedules nothing on the day it was read in. */
          stopsOn: Stopped
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
  /**
   * Cards answered under it since the day opened, and the minutes they took.
   * The two beside the total divide it the way a budget does, so each is
   * weighed against the budget of its own kind.
   */
  readonly answered: number
  readonly answeredNew: number
  readonly answeredReviews: number
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

  /**
   * Whether the presets of that vault have been read. A deck the reading passed
   * over is a deck whose preset could not be read, and until it has run nothing
   * is known either way.
   */
  const known = ref(false)

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
    known.value = false
  }

  // Today is the review day, which the application measures and this window is
  // told: it begins at the hour the settings name.
  const read = async (vault: Owing | null, today: string) => {
    if (!vault) {
      forget()
      return
    }
    // A vault read again keeps what is known of it while the reading runs, so
    // the screen it is read behind does not empty and fill.
    if (of.value !== vault.vaultId) known.value = false
    of.value = vault.vaultId

    const held = await Promise.all(
      vault.decks.map((deck) => scheduled(deps.presets, vault.vaultId, deck.deck)),
    )
    if (of.value !== vault.vaultId) return

    presets.value = gather(vault, held, today)
    known.value = true
  }

  return { presets, byDeck, of, known, read, forget }
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
    stopsOn: Stopped
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
        settings: { ...settings, goal: goaled[settings.goal] ?? 'minutes' },
        problems: answer.preset.problems,
        stopsOn: answer.preset.stopsOn,
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
  stopsOn: Stopped
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
        stopsOn: one.held.stopsOn,
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
      answeredNew: day?.answeredNew ?? 0,
      answeredReviews: day?.answeredReviews ?? 0,
      took: day?.took ?? 0,
      paused: stoppedWords(one.stopsOn, one.settings, today),
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
      answeredNew: one.answeredNew,
      answeredReviews: one.answeredReviews,
      took: one.took,
      // The count answered for this preset, so its verdict is the count's.
      paused: stoppedWords(one.stopsOn, null, today),
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
 */
export const opens = (deck: DeckOwing, by: ReadonlyMap<string, Preset>): boolean =>
  deck.due + deck.new > 0 && !by.get(deck.deck)?.paused

/**
 * Whether a deck holds nothing that can ever be asked for: every card face in
 * it is one nobody has begun, and the preset scheduling it begins none a day.
 *
 * It is a fact about the material, and not a reason the preset is stopped. The
 * preset schedules; there is nothing here for it to schedule.
 */
export const beginsNothing = (deck: DeckOwing, by: Preset | undefined): boolean =>
  deck.faces > 0 && deck.unbegun === deck.faces && by?.budget.new === 0

/** How many cards the day holds at most. */
export const holds = (budget: Budget): number => budget.new + budget.reviews

/**
 * How far through its day a preset stands: what has been answered against the
 * cards the day holds, and what it has taken against the minutes the day runs,
 * whichever of the two is further along.
 *
 * A day answered past what its budget holds stands above one.
 */
export const through = (one: Preset): number => {
  // Only a budget that closes the day is weighed against, and each is weighed
  // against what was answered of its own kind. A preset steered by its minutes
  // keeps its card counts as the person left them.
  const shares = [
    one.closes.new ? share(one.answeredNew, one.budget.new) : 0,
    one.closes.reviews ? share(one.answeredReviews, one.budget.reviews) : 0,
    one.closes.minutes ? share(one.took, one.budget.minutes) : 0,
  ]
  return Math.max(0, ...shares)
}

/** What a count comes to against a budget. A budget of nothing is no share. */
const share = (spent: number, budget: number): number => (budget > 0 ? spent / budget : 0)

/**
 * Whether a preset's day is spent, which is what leaves every deck under it
 * nothing more to ask however much those decks still hold.
 */
export const spent = (one: Preset): boolean => through(one) >= 1

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
  passed: (day: string) => `${dayWords(day)} has passed`,
  noLoad: (day: string) => `no load on ${weekdayWords(day)}`,
  /** No day of the week carries any of the load, so there is no next day. */
  noWeek: 'no load on any day',
  /** Every card face here is unbegun, and the preset begins none a day. */
  beginsNothing: 'no cards to begin',
} as const

/**
 * How much of a deck stands learned, and what is said where no share can be.
 *
 * The word stands with the figure.
 */
export const LEARNED = {
  share: (of: number) => `${Math.round(of * 100)}% learned`,
  /** The preset scheduling the deck could not be read, and the rule is its. */
  unruled: 'no rule to count by',
} as const

/**
 * How much of a deck stands learned, as a share of its card faces, and null for
 * a deck holding none: a share of nothing is no share.
 */
export const learned = (deck: DeckOwing): number | null =>
  deck.faces > 0 ? deck.learned / deck.faces : null

/**
 * How many cards sitting down to this preset would put in front of a person,
 * which is what its decks still owe today, or why it would put none there.
 *
 * It is the count the sitting itself will ask, so it is printed as it stands.
 */
export const leftWords = (one: Preset): string => {
  if (one.cards > 0) return many(one.cards, 'card')
  return spent(one) ? STOPPED.full : STOPPED.nothing
}

/**
 * The words each verdict the vault may hand over comes to. A verdict scheduling
 * something says nothing. Every value stands here, so a verdict added to the
 * schema is one this window is made to answer.
 */
const WHY: Record<Stopped, (settings: Settings | null, today: string) => string> = {
  [Stopped.UNSPECIFIED]: () => '',
  [Stopped.NOTHING]: () => '',
  [Stopped.NO_MINUTES]: () => STOPPED.noMinutes,
  [Stopped.NO_CARDS]: () => STOPPED.noCards,
  [Stopped.NO_DAY]: () => STOPPED.noDay,
  [Stopped.PAST_DAY]: (settings) =>
    settings?.byDate ? STOPPED.passed(settings.byDate) : STOPPED.pastDay,
  [Stopped.NO_LOAD]: (settings, today) => STOPPED.noLoad(today),
  [Stopped.NO_WEEK]: () => STOPPED.noWeek,
}

/**
 * Why a preset schedules nothing today, in the words to show, and empty while
 * it schedules something.
 *
 * The verdict is the core's: it is what the sitting hands its cards out by. Two
 * of the reasons name a day, and the settings carry the one a date aimed at.
 */
export const stoppedWords = (why: Stopped, settings: Settings | null, today: string): string =>
  WHY[why](settings, today)

/**
 * What the goal of a preset comes to, in the few words a person reads at a
 * glance. A day is said as a person reads one, and not as the file writes it.
 */
export const goalWords = (settings: Settings, today: string): string => {
  switch (settings.goal) {
    case 'retention':
      return `${Math.round(settings.retention * 100)}% remembered`
    case 'date': {
      if (!settings.byDate) return 'by no day'
      const left = daysBetween(today, settings.byDate)
      if (left <= 0) return `by ${dayWords(settings.byDate)}`
      return `${many(left, 'day')} to ${dayWords(settings.byDate)}`
    }
    case 'minutes':
      if (settings.minutesADay === 0) return STOPPED.noMinutes
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
