/**
 * Asking the application about the presets of the vault it is showing.
 *
 * A preset is a note, and its settings are frontmatter keys; nothing here
 * reads a file. What travels is the settings, the file they came out of, and
 * what the whole range of the goal comes to.
 */
import { createClient } from '@connectrpc/connect'
import {
  BudgetName,
  Counts as Countings,
  Goal as Goals,
  Rule as Rules,
  PresetsService,
} from '@numen/protocol'
import type {
  StopReason,
  Bounds as BoundsMessage,
  Curve as CurveMessage,
  Place as PlaceMessage,
  Preset as PresetMessage,
  Settings as SettingsMessage,
  SettingsBounds as SettingsBoundsMessage,
} from '@numen/protocol'
import { namesOf, transport } from '@numen/wire'
import { fingerprint, refusalIn, staleIn, stamp } from '../answers'
import type { RefusalReason } from '../core'

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
 * What a preset counts as learned. It is the person's rule for when the
 * material is theirs.
 *
 * Under `interval` a card face is learned once it is sent away for that many
 * days; under `retention` once the chance of recalling it today stands at the
 * target. The rule not named keeps its value and takes no part.
 */
export type Rule = 'interval' | 'retention'

/** The two, in the order they are offered. */
export const RULES: readonly Rule[] = ['interval', 'retention']

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
  /**
   * What share of a day's cards goes to what is overdue before any new
   * material is offered, as a percentage. It moves what fills a day and never
   * how much a day holds: a hundred works the debt down first, nothing offers
   * new material while there is any, and between them the day is split until
   * one side runs out.
   */
  readonly backlog: number
  /**
   * How much of a day's load each day of the week carries, in per cent, under
   * the first three letters of the day's name. A day not named carries the
   * whole of it, and a day at nothing schedules nothing.
   */
  readonly load: Load
  readonly evenLoad: boolean
  /** What counts as learned, and how long a card is sent away for under one. */
  readonly learned: Rule
  readonly interval: number
}

/** A share of a day's load for each day of the week that is not at the whole. */
export type Load = Readonly<Record<string, number>>

/** The whole of a day's load, which a day nothing was said about carries. */
export const WHOLE_LOAD = 100

/** The shares of a day's load a day may be put at, in the order they are offered. */
export const LOADS: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE_LOAD]

/** What one day of the week carries, which is the whole of it unless it is named. */
export const loadOn = (load: Load, day: string): number => load[day] ?? WHOLE_LOAD

/**
 * The load with one day put at a share. What carries the whole of a day is what
 * nothing was said about, so a day put back to it stops being named.
 */
export const loaded = (load: Load, day: string, share: number): Load => {
  const out: Record<string, number> = { ...load }
  if (share === WHOLE_LOAD) delete out[day]
  else out[day] = share
  return out
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
  backlog: 100,
  load: {},
  evenLoad: true,
  learned: 'interval',
  interval: 21,
}

/** How far one setting goes, at each end. A number outside them is refused. */
export interface Bounds {
  readonly least: number
  readonly most: number
}

/**
 * How far each setting goes, as a read answers it. They are the application's
 * and not the preset's, so every read carries the same ones.
 *
 * A setting the application said no bound for is absent here, and goes as far
 * as the field a person types it into does.
 */
export interface SettingsBounds {
  readonly minutesADay?: Bounds
  readonly newADay?: Bounds
  readonly reviewsADay?: Bounds
  readonly retention?: Bounds
  readonly backlog?: Bounds
  readonly interval?: Bounds
}

/** How far each setting goes until the application has said. */
export const NO_BOUNDS: SettingsBounds = {}

/** One preset as a read hands it over. */
export interface Preset {
  /** The note the settings were read from. Empty for a deck naming no preset. */
  readonly path: string
  readonly title: string
  readonly settings: Settings
  /** What was wrong in the file and was not guessed at, in the words to show. */
  readonly problems: readonly string[]
  /** Why it schedules nothing at all, which holds on every day. */
  readonly stops: StopReason
  /**
   * The same asked of the day holding now. A day of the week carrying none of
   * the load is said here alone.
   */
  readonly stopsOn: StopReason
}

/** What reading a preset came back with. */
export interface ReadResult {
  /** Null when the preset was refused. */
  readonly preset: Preset | null
  readonly refusal: RefusalReason | null
  /** The file it came out of, to present at the next write. */
  readonly at: string
  /** How far each setting goes, which a refused read answers as well. */
  readonly bounds: SettingsBounds
}

/** What writing a preset came back with. */
export interface WriteResult {
  readonly refusal: RefusalReason | null
  /** The file is no longer the one this caller read, and nothing was written. */
  readonly changed: boolean
  readonly at: string
}

/** What making a preset came back with. */
export interface MakeResult {
  /** Where it is filed. Empty when nothing was made. */
  readonly path: string
  readonly refusal: RefusalReason | null
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
  /**
   * Every budget that closed the day here, as the schema names them. None is a
   * day that asked for every card there was, and a day held to two counts names
   * both.
   */
  readonly closed: readonly BudgetName[]
  /**
   * How many days of review at this place before nothing is overdue. Zero is a
   * preset standing over nothing overdue, and -1 is a pace that never gets
   * there.
   */
  readonly clears: number
  /** How many card faces stand learned as the run opens, under the preset's rule. */
  readonly learned: number
  /**
   * How many days of review before every card face is learned. Zero is a place
   * standing over a material already learned, and -1 is a horizon that ends
   * with one of them still to learn.
   *
   * There is no such day under every rule: a target is a level a card drifts
   * in and out of, and a date is the day itself. Where there is none, none is
   * carried and nothing is said in its place.
   */
  readonly learns?: number
  /**
   * How many card faces cannot be learned by the day named whatever the pace:
   * the rule wants more days than the day leaves them. It is a count and not a
   * pace, so no budget moves it.
   */
  readonly short: number
  /**
   * How many card faces stand overdue at the end of each day projected here,
   * one entry a day. It runs over days, which is a different axis from the
   * grid.
   */
  readonly backlog: readonly number[]
}

/** One place on the curve worth pointing at. */
export interface Place {
  /** Where on the grid it stands, and -1 for a value that falls outside it. */
  readonly at: number
  readonly value: number
  /** The day at the place, filled for a goal of a date. */
  readonly day: string
}

/** A place that falls outside the grid. */
export const NOWHERE: Place = { at: -1, value: 0, day: '' }

/** What the one control comes to over the whole range of its goal. */
/**
 * What a preset schedules, as the figures over the picture count it. No setting
 * moves one of them.
 */
export interface PresetCounts {
  /** How many decks are scheduled by this preset. */
  readonly decks: number
  /** How many card faces stand in those decks. */
  readonly cards: number
  /**
   * How many of those card faces have had their day and were not answered on
   * it. It is the backlog alone: what falls due today is not part of it.
   */
  readonly overdue: number
  /**
   * How many of those card faces nobody has answered at all, so they have had
   * no day. It never overlaps the overdue.
   */
  readonly unbegun: number
}

export interface Curve extends PresetCounts {
  readonly goal: Goal
  /** The goal's value at each place: minutes, a share of cards, or days from today. */
  readonly grid: readonly number[]
  /** The day of each place, filled for a goal of a date. */
  readonly days: readonly string[]
  readonly at: readonly Point[]
  readonly now: Place
  readonly suggested: Place
  /**
   * Whether this is the application's answer. A curve the window worked out
   * for itself stands until that answer lands.
   */
  readonly honest: boolean
}

/** One preset as a person choosing between them sees it. */
export interface PresetChoice {
  readonly path: string
  /** What it is called. Empty where nothing names the note. */
  readonly title: string
}

/** What the window asks about the presets of a vault. */
export interface Presets {
  /** The settings of one preset, and the file they came out of. */
  read(path: string): Promise<ReadResult>
  /**
   * Every preset the vault holds. The defaults are no note and are not among
   * them: they are what schedules a deck naming no preset.
   */
  list(): Promise<readonly PresetChoice[]>
  /**
   * A preset made in a folder under the name it is given, naming none of its
   * settings. Every key it does not carry stands at the default.
   */
  makes(title: string, folder: string): Promise<MakeResult>
  /**
   * A deck put on a preset, and on the defaults where the path is empty. Seen
   * is what a read of the deck gave this caller, and a deck that moved past it
   * comes back changed with nothing written.
   */
  schedules(deck: string, preset: string, seen: string): Promise<WriteResult>
  /** The preset a deck is scheduled by. A deck naming none answers under no path. */
  scheduling(deck: string): Promise<ReadResult>
  /**
   * Settings into a preset. Seen is what a read gave this caller, and a file
   * that moved past it comes back changed with nothing written.
   */
  write(path: string, settings: Settings, seen: string): Promise<WriteResult>
  /**
   * What those settings come to over the whole range of the goal they name.
   * Nothing is written: a curve is asked for what a person is still moving.
   */
  curve(path: string, settings: Settings): Promise<Curve>
}

const asking = createClient(PresetsService, transport)

/** The same questions, in the shape the window asks them. */
export const presets: Presets = {
  read: async (path) => took(await asking.readPreset({ path })),
  scheduling: async (deck) => took(await asking.getDeckPreset({ deck })),
  list: async () => (await asking.listPresets({})).presets.map(
    (one) => ({ path: one.path, title: one.title }),
  ),
  makes: async (title, folder) => {
    const answer = await asking.createPreset({ title, folder })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  schedules: async (deck, preset, seen) => {
    const answer = await asking.scheduleDeck({
      deck,
      preset,
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { refusal: refusalIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  write: async (path, settings, seen) => {
    const answer = await asking.writePreset({
      path,
      settings: sent(settings),
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { refusal: refusalIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  curve: async (path, settings) => {
    const answer = await asking.computeCurve({ path, settings: sent(settings) })
    return curved(answer.curve)
  },
}

/** What a read answered, whichever of the two asked it. */
const took = (answer: {
  preset?: PresetMessage | undefined
  refusal?: number | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  bounds?: SettingsBoundsMessage | undefined
}): ReadResult => ({
  preset: answer.preset ? held(answer.preset) : null,
  refusal: refusalIn(answer),
  at: stamp(answer.at) ?? '',
  bounds: bounded(answer.bounds),
})

/** How far each setting goes, as the read answered it. */
const bounded = (said: SettingsBoundsMessage | undefined): SettingsBounds => {
  const out: { -readonly [field in keyof SettingsBounds]: Bounds } = {}
  if (said === undefined) return out
  if (said.minutesADay) out.minutesADay = ranged(said.minutesADay)
  if (said.newADay) out.newADay = ranged(said.newADay)
  if (said.reviewsADay) out.reviewsADay = ranged(said.reviewsADay)
  if (said.retention) out.retention = ranged(said.retention)
  if (said.backlog) out.backlog = ranged(said.backlog)
  if (said.interval) out.interval = ranged(said.interval)
  return out
}

/** One pair of ends, in the window's own words. */
const ranged = (said: BoundsMessage): Bounds => ({ least: said.least, most: said.most })

/** One preset as the window carries it. */
const held = (one: PresetMessage): Preset => ({
  path: one.path,
  title: one.title,
  settings: settingsOf(one.settings),
  problems: one.problems,
  stops: one.stops,
  stopsOn: one.stopsOn,
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
        backlog: said.backlog,
        load: said.load,
        evenLoad: said.evenLoad,
        learned: LEARNED[said.learned] ?? DEFAULTS.learned,
        interval: said.interval,
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
  backlog: settings.backlog,
  load: { ...settings.load },
  evenLoad: settings.evenLoad,
  learned: RULING[settings.learned],
  interval: settings.interval,
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
    closed: one.closed,
    clears: one.clears,
    learned: one.learned,
    // A day the whole of it stands learned on, where the rule has one.
    ...(one.learns === undefined ? {} : { learns: one.learns }),
    short: one.short,
    backlog: one.backlog,
  })),
  now: placed(said?.now),
  suggested: placed(said?.suggested),
  decks: said?.decks ?? 0,
  cards: said?.cards ?? 0,
  overdue: said?.overdue ?? 0,
  unbegun: said?.unbegun ?? 0,
  honest: true,
})

const placed = (said: PlaceMessage | undefined): Place =>
  said === undefined ? NOWHERE : { at: said.at, value: said.value, day: said.day }

/**
 * The goal in the window's own words. A preset naming none takes the default.
 * Keyed by the schema, so a goal added to it has to be given a word here before
 * this compiles.
 */
const WORDED: Record<Goals, Goal | null> = {
  [Goals.UNSPECIFIED]: null,
  [Goals.MINUTES_A_DAY]: 'minutes',
  [Goals.RETENTION]: 'retention',
  [Goals.BY_DATE]: 'date',
}

/** The goal as the schema names it, read off the words above. */
const ASKED = namesOf<Goal, Goals>(WORDED)

/**
 * What counts as learned, in the window's own words. A preset naming none takes
 * the default. Keyed by the schema, the same way.
 */
const LEARNED: Record<Rules, Rule | null> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

/** What counts as learned, as the schema names it. */
const RULING = namesOf<Rule, Rules>(LEARNED)

/**
 * What a budget counts, in the window's own words. A preset naming nothing
 * takes the default. Keyed by the schema, the same way.
 */
const COUNTED: Record<Countings, Counts | null> = {
  [Countings.UNSPECIFIED]: null,
  [Countings.CARDS]: 'cards',
  [Countings.SHOWS]: 'shows',
}

/** What a budget counts, as the schema names it. */
const COUNTING = namesOf<Counts, Countings>(COUNTED)
