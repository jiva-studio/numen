/**
 * Asking the application about the presets of the vault it is showing.
 *
 * A preset is a note, and its settings are frontmatter keys; nothing here
 * reads a file. What travels is the settings, the file they came out of, and
 * what the whole range of the goal comes to.
 */
import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { Counts as Countings, Goal as Goals, PresetsService } from '@numen/protocol'
import type {
  Curve as CurveMessage,
  Mark as MarkMessage,
  Preset as PresetMessage,
  Settings as SettingsMessage,
} from '@numen/protocol'
import { fingerprint, refusalIn, stamp } from '../answers'
import type { Refused } from '../core'

/** Which value the one control steers. */
export type Goal = 'minutes' | 'retention' | 'date'

/** The three, in the order they are offered. */
export const GOALS: readonly Goal[] = ['minutes', 'retention', 'date']

/**
 * What a day's budget is spent on. Under `cards` a card face is charged the
 * first time it is answered in a review day and comes round again in that day
 * for nothing; under `shows` every showing is charged.
 */
export type Counts = 'cards' | 'shows'

/** The two, in the order they are offered. */
export const COUNTS: readonly Counts[] = ['cards', 'shows']

/**
 * How the decks pointing at one preset are scheduled. A preset carrying none
 * of these keys is the defaults, and no cards a day is a pause.
 */
export interface Settings {
  readonly goal: Goal
  /** The day the material is to be in the head, as `2026-09-30`. Empty for none. */
  readonly byDate: string
  readonly minutesADay: number
  readonly newADay: number
  readonly reviewsADay: number
  readonly retention: number
  /** What a day's budget is spent on. */
  readonly counts: Counts
  /** The days the load is cut on, each as the first three letters of its name. */
  readonly lightDays: readonly string[]
  readonly evenLoad: boolean
}

/** A preset naming nothing, and how a deck pointing at none is scheduled. */
export const DEFAULTS: Settings = {
  goal: 'minutes',
  byDate: '',
  minutesADay: 20,
  newADay: 10,
  reviewsADay: 200,
  retention: 0.9,
  counts: 'cards',
  lightDays: [],
  evenLoad: true,
}

/** How far each setting goes. A number outside its bounds is refused. */
export const BOUNDS = {
  minutesADay: { least: 0, most: 24 * 60 },
  newADay: { least: 0, most: 9999 },
  reviewsADay: { least: 0, most: 9999 },
  retention: { least: 0.7, most: 0.99 },
} as const

/** One preset as a read hands it over. */
export interface Preset {
  /** The note the settings were read from. Empty for a deck naming no preset. */
  readonly path: string
  readonly title: string
  readonly settings: Settings
  /** What was wrong in the file and was not guessed at, in the words to show. */
  readonly problems: readonly string[]
}

/** What reading a preset came back with. */
export interface Read {
  /** Null when the preset was refused. */
  readonly preset: Preset | null
  readonly refusal: Refused | null
  /** The file it came out of, to present at the next write. */
  readonly at: string
}

/** What writing a preset came back with. */
export interface Written {
  readonly refusal: Refused | null
  /** The file is no longer the one this caller read, and nothing was written. */
  readonly changed: boolean
  readonly at: string
}

/** What a preset comes to at one place of the grid. */
export interface Point {
  readonly reviews: number
  readonly minutes: number
  /** The share of the material that comes back. */
  readonly retained: number
  /** The backlog the budget did not carry. */
  readonly owed: number
  /** The share got through by this day, and whether a budget gets through it. */
  readonly through: number
  readonly enough: boolean
  readonly met: boolean
}

/** One place on the curve worth pointing at. */
export interface Mark {
  /** Where on the grid it stands, and -1 for a value that falls outside it. */
  readonly at: number
  readonly value: number
  /** The day at the mark, filled for a goal of a date. */
  readonly day: string
}

/** A mark that falls outside the grid. */
export const NOWHERE: Mark = { at: -1, value: 0, day: '' }

/** What the one control comes to over the whole range of its goal. */
export interface Curve {
  readonly goal: Goal
  /** The goal's value at each place: minutes, a share of cards, or days from today. */
  readonly grid: readonly number[]
  /** The day of each place, filled for a goal of a date. */
  readonly days: readonly string[]
  readonly at: readonly Point[]
  readonly now: Mark
  readonly suggested: Mark
  /** How many decks are scheduled by this preset. */
  readonly decks: number
  /**
   * Whether this is the application's answer. A curve the window worked out
   * for itself stands until that answer lands.
   */
  readonly honest: boolean
}

/** What the window asks about the presets of a vault. */
export interface Presets {
  /** The settings of one preset, and the file they came out of. */
  read(path: string): Promise<Read>
  /** The preset a deck is scheduled by. A deck naming none answers under no path. */
  scheduling(deck: string): Promise<Read>
  /**
   * Settings into a preset. Seen is what a read gave this caller, and a file
   * that moved past it comes back changed with nothing written.
   */
  write(path: string, settings: Settings, seen: string): Promise<Written>
  /**
   * What those settings come to over the whole range of the goal they name.
   * Nothing is written: a curve is asked for what a person is still moving.
   */
  curve(path: string, settings: Settings): Promise<Curve>
}

const asking = createClient(
  PresetsService,
  createConnectTransport({ baseUrl: window.location.origin }),
)

/** The same questions, in the shape the window asks them. */
export const presets: Presets = {
  read: async (path) => took(await asking.readPreset({ path })),
  scheduling: async (deck) => took(await asking.scheduling({ deck })),
  write: async (path, settings, seen) => {
    const answer = await asking.writePreset({
      path,
      settings: sent(settings),
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { refusal: refusalIn(answer), changed: answer.changed, at: stamp(answer.at) ?? '' }
  },
  curve: async (path, settings) => {
    const answer = await asking.curve({ path, settings: sent(settings) })
    return curved(answer.curve)
  },
}

/** What a read answered, whichever of the two asked it. */
const took = (answer: {
  preset?: PresetMessage | undefined
  refusal?: number | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
}): Read => ({
  preset: answer.preset ? held(answer.preset) : null,
  refusal: refusalIn(answer),
  at: stamp(answer.at) ?? '',
})

/** One preset as the window carries it. */
const held = (one: PresetMessage): Preset => ({
  path: one.path,
  title: one.title,
  settings: settingsOf(one.settings),
  problems: one.problems,
})

/** The settings in the window's own words. A preset carrying none is the defaults. */
const settingsOf = (said: SettingsMessage | undefined): Settings =>
  said === undefined
    ? DEFAULTS
    : {
        goal: WORDED[said.goal] ?? DEFAULTS.goal,
        byDate: said.byDate,
        minutesADay: said.minutesADay,
        newADay: said.newADay,
        reviewsADay: said.reviewsADay,
        retention: said.retention,
        counts: COUNTED[said.counts] ?? DEFAULTS.counts,
        lightDays: said.lightDays,
        evenLoad: said.evenLoad,
      }

/** The settings in the shape the schema carries them. */
const sent = (settings: Settings) => ({
  goal: ASKED[settings.goal],
  byDate: settings.byDate,
  minutesADay: settings.minutesADay,
  newADay: settings.newADay,
  reviewsADay: settings.reviewsADay,
  retention: settings.retention,
  counts: COUNTING[settings.counts],
  lightDays: [...settings.lightDays],
  evenLoad: settings.evenLoad,
})

/** A curve as the window carries it. An answer holding none is an empty one. */
const curved = (said: CurveMessage | undefined): Curve => ({
  goal: (said && WORDED[said.goal]) ?? DEFAULTS.goal,
  grid: said?.grid ?? [],
  days: said?.days ?? [],
  at: (said?.at ?? []).map((one) => ({
    reviews: one.reviews,
    minutes: one.minutes,
    retained: one.retained,
    owed: one.owed,
    through: one.through,
    enough: one.enough,
    met: one.met,
  })),
  now: marked(said?.now),
  suggested: marked(said?.suggested),
  decks: said?.decks ?? 0,
  honest: true,
})

const marked = (said: MarkMessage | undefined): Mark =>
  said === undefined ? NOWHERE : { at: said.at, value: said.value, day: said.day }

/** The goal as the schema names it. */
const ASKED: Record<Goal, Goals> = {
  minutes: Goals.MINUTES_A_DAY,
  retention: Goals.RETENTION,
  date: Goals.BY_DATE,
}

/** The goal in the window's own words. */
const WORDED: Partial<Record<Goals, Goal>> = {
  [Goals.MINUTES_A_DAY]: 'minutes',
  [Goals.RETENTION]: 'retention',
  [Goals.BY_DATE]: 'date',
}

/** What a budget counts, as the schema names it. */
const COUNTING: Record<Counts, Countings> = {
  cards: Countings.CARDS,
  shows: Countings.SHOWS,
}

/** What a budget counts, in the window's own words. */
const COUNTED: Partial<Record<Countings, Counts>> = {
  [Countings.CARDS]: 'cards',
  [Countings.SHOWS]: 'shows',
}
