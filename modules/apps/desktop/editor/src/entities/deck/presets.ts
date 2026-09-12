/**
 * ConnectRPC client and domain models for flashcard preset services.
 */
import {
  BudgetUnit as BudgetUnits,
  Rule as Rules,
  type BudgetName,
  type Bounds as BoundsMessage,
  type Curve as CurveMessage,
  type Place as PlaceMessage,
  type Preset as PresetMessage,
  type Settings as SettingsMessage,
  type SettingsBounds as SettingsBoundsMessage,
  type StopReason,
} from '@numen/protocol'
import { goalNames, goalOf, namesOf, type Goal } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '@/shared/answers'
import { asking } from '@/shared/clients'
import type { ErrorCode } from '@/shared/errors'

export type { Goal }

/** The three, in the order they are offered. */
export const GOALS: readonly Goal[] = ['minutes', 'retention', 'date']

/**
 * The unit a day's budget is spent in.
 */
export type BudgetUnit = 'cards' | 'shows'

/** The two, in the order they are offered. */
export const BUDGET_UNITS: readonly BudgetUnit[] = ['cards', 'shows']

/**
 * What a preset counts as learned.
 */
export type Rule = 'interval' | 'retention'

/** The two, in the order they are offered. */
export const RULES: readonly Rule[] = ['interval', 'retention']

/** A share of a day's load for each day of the week that is not at the whole. */
export type Load = Readonly<Record<string, number>>

/** The whole of a day's load, which a day nothing was said about carries. */
export const WHOLE_LOAD = 100

/** The shares of a day's load a day may be put at, in the order they are offered. */
export const LOADS: readonly number[] = [0, 10, 25, 50, 75, 90, WHOLE_LOAD]

export const loadOn = (load: Load, day: string): number => load[day] ?? WHOLE_LOAD

/**
 * The load with one day put at a share.
 */
export const setLoadOn = (load: Load, day: string, share: number): Load => {
  const out: Record<string, number> = { ...load }
  if (share === WHOLE_LOAD) delete out[day]
  else out[day] = share
  return out
}

/**
 * How the decks pointing at one preset are scheduled.
 */
export interface Settings {
  readonly goal: Goal
  /** The day the material is to be in the head, as `2026-09-30`. Empty for none. */
  readonly byDate: string
  readonly minutesADay: number
  readonly newADay: number
  readonly reviewsADay: number
  readonly retention: number
  /** The unit a day's budget is spent in. */
  readonly counts: BudgetUnit
  readonly backlog: number
  readonly load: Load
  readonly evenLoad: boolean
  readonly learned: Rule
  readonly interval: number
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

/** How far one setting goes, at each end. */
export interface Bounds {
  readonly least: number
  readonly most: number
}

/**
 * How far each setting goes, as a read answers it.
 */
export interface SettingsBounds {
  readonly minutesADay?: Bounds
  readonly newADay?: Bounds
  readonly reviewsADay?: Bounds
  readonly retention?: Bounds
  readonly backlog?: Bounds
  readonly interval?: Bounds
  readonly load?: Bounds
}

/** How far each setting goes until the application has said. */
export const NO_BOUNDS: SettingsBounds = {}

/** One preset as a read hands it over. */
export interface Preset {
  readonly path: string
  readonly title: string
  readonly settings: Settings
  readonly problems: readonly string[]
  readonly stops: StopReason
  readonly stopsOn: StopReason
}

/** What reading a preset came back with. */
export interface ReadResult {
  readonly preset: Preset | null
  readonly error: ErrorCode | null
  readonly at: string
  readonly bounds: SettingsBounds
}

/** What writing a preset came back with. */
export interface WriteResult {
  readonly error: ErrorCode | null
  readonly changed: boolean
  readonly at: string
}

/** What making a preset came back with. */
export interface MakeResult {
  readonly path: string
  readonly error: ErrorCode | null
}

/** What a preset comes to at one place of the grid. */
export interface Point {
  readonly reviews: number
  readonly minutes: number
  readonly retained: number
  readonly owed: number
  readonly through: number
  readonly enough: boolean
  readonly closed: readonly BudgetName[]
  readonly clears: number
  readonly learned: number
  readonly learns?: number
  readonly short: number
  readonly backlog: readonly number[]
}

/** One place on the curve worth pointing at. */
export interface Place {
  readonly at: number
  readonly value: number
  readonly day: string
}

/** A place that falls outside the grid. */
export const NOWHERE: Place = { at: -1, value: 0, day: '' }

/**
 * What a preset schedules, as the figures over the picture count it.
 */
export interface PresetCounts {
  readonly decks: number
  readonly cards: number
  readonly overdue: number
  readonly unbegun: number
}

export interface Curve extends PresetCounts {
  readonly goal: Goal
  readonly grid: readonly number[]
  readonly days: readonly string[]
  readonly at: readonly Point[]
  readonly now: Place
  readonly suggested: Place
  readonly isValid?: boolean
  readonly honest: boolean
}

/** One preset as a person choosing between them sees it. */
export interface PresetChoice {
  readonly path: string
  readonly title: string
}

/** What the window asks about the presets of a vault. */
export interface Presets {
  read(path: string): Promise<ReadResult>
  list(): Promise<readonly PresetChoice[]>
  makes(title: string, folder: string): Promise<MakeResult>
  schedules(deck: string, preset: string, seen: string): Promise<WriteResult>
  scheduling(deck: string): Promise<ReadResult>
  write(path: string, settings: Settings, seen: string): Promise<WriteResult>
  curve(path: string, settings: Settings): Promise<Curve>
}

/** The same questions, in the shape the window asks them. */
export const presets: Presets = {
  read: async (path) => took(await asking.readPreset({ path })),
  scheduling: async (deck) => took(await asking.getDeckPreset({ deck })),
  list: async () =>
    (await asking.listPresets({})).presets.map((one) => ({ path: one.path, title: one.title })),
  makes: async (title, folder) => {
    const answer = await asking.createPreset({ title, path: folder })
    return { path: answer.path, error: errorIn(answer) }
  },
  schedules: async (deck, preset, seen) => {
    const answer = await asking.scheduleDeck({
      deck,
      preset,
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { error: errorIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  write: async (path, settings, seen) => {
    const answer = await asking.writePreset({
      path,
      settings: sent(settings),
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { error: errorIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  curve: async (path, settings) => {
    const answer = await asking.computeCurve({ path, settings: sent(settings) })
    return parseCurve(answer.curve)
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
  error: errorIn(answer),
  at: stamp(answer.at) ?? '',
  bounds: parseSettingsBounds(answer.bounds),
})

/** How far each setting goes, as the read answered it. */
const parseSettingsBounds = (said: SettingsBoundsMessage | undefined): SettingsBounds => {
  const out: { -readonly [field in keyof SettingsBounds]: Bounds } = {}
  if (said === undefined) return out
  if (said.minutesADay) out.minutesADay = parseBounds(said.minutesADay)
  if (said.newADay) out.newADay = parseBounds(said.newADay)
  if (said.reviewsADay) out.reviewsADay = parseBounds(said.reviewsADay)
  if (said.retention) out.retention = parseBounds(said.retention)
  if (said.backlog) out.backlog = parseBounds(said.backlog)
  if (said.interval) out.interval = parseBounds(said.interval)
  if (said.load) out.load = parseBounds(said.load)
  return out
}

/** One pair of ends, in the window's own words. */
const parseBounds = (said: BoundsMessage): Bounds => ({ least: said.least, most: said.most })

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
        goal: goalOf[said.goal] ?? DEFAULTS.goal,
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
  goal: goalNames[settings.goal],
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
const parseCurve = (said: CurveMessage | undefined): Curve => ({
  goal: (said && goalOf[said.goal]) ?? DEFAULTS.goal,
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
    ...(one.learns === undefined ? {} : { learns: one.learns }),
    short: one.short,
    backlog: one.backlog,
  })),
  now: parsePlace(said?.now),
  suggested: parsePlace(said?.suggested),
  decks: said?.decks ?? 0,
  cards: said?.cards ?? 0,
  overdue: said?.overdue ?? 0,
  unbegun: said?.unbegun ?? 0,
  isValid: true,
  honest: true,
})

const parsePlace = (said: PlaceMessage | undefined): Place =>
  said === undefined ? NOWHERE : { at: said.at, value: said.value, day: said.day }

/**
 * What counts as learned, in the window's own words.
 */
const LEARNED: Record<Rules, Rule | null> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

const RULING = namesOf<Rule, Rules>(LEARNED)

/**
 * The unit a budget is spent in, in the window's own words.
 */
const COUNTED: Record<BudgetUnits, BudgetUnit | null> = {
  [BudgetUnits.UNSPECIFIED]: null,
  [BudgetUnits.CARDS]: 'cards',
  [BudgetUnits.SHOWS]: 'shows',
}

const COUNTING = namesOf<BudgetUnit, BudgetUnits>(COUNTED)
