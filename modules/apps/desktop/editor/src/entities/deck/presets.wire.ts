/** The presets of a vault, as the window asks for them and as the schema writes them. */
import {
  BudgetUnit as BudgetUnits,
  Rule as Rules,
  type Bounds as BoundsMessage,
  type Curve as CurveMessage,
  type Place as PlaceMessage,
  type Preset as PresetMessage,
  type Settings as SettingsMessage,
  type SettingsBounds as SettingsBoundsMessage,
} from '@numen/protocol'
import { goalNames, goalOf, namesOf } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '@/shared/answers'
import { asking } from '@/shared/clients'
import { DEFAULTS, NOWHERE } from './presets'
import type {
  Bounds,
  BudgetUnit,
  Curve,
  Place,
  Preset,
  Presets,
  ReadResult,
  Rule,
  Settings,
  SettingsBounds,
} from './presets'

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
