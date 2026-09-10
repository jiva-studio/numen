/**
 * ConnectRPC client and adapter for flashcard preset services.
 */
import { createClient } from '@connectrpc/connect'
import {
  BudgetName,
  BudgetUnit as BudgetUnits,
  Rule as Rules,
  PresetsService,
} from '@numen/protocol'
import type {
  Bounds as BoundsMessage,
  Curve as CurveMessage,
  Place as PlaceMessage,
  Preset as PresetMessage,
  Settings as SettingsMessage,
  SettingsBounds as SettingsBoundsMessage,
} from '@numen/protocol'
import { goalNames, goalOf, namesOf, transport } from '@numen/wire'
import { fingerprint, refusalIn, staleIn, stamp } from '../shared/answers'
import {
  DEFAULTS,
  NOWHERE,
  type Bounds,
  type BudgetUnit,
  type Curve,
  type MakeResult,
  type Place,
  type Preset,
  type Presets,
  type ReadResult,
  type Rule,
  type Settings,
  type SettingsBounds,
  type WriteResult,
} from './types'

export * from './types'

const asking = createClient(PresetsService, transport)

/** The same questions, in the shape the window asks them. */
export const presets: Presets = {
  read: async (path) => took(await asking.readPreset({ path })),
  scheduling: async (deck) => took(await asking.getDeckPreset({ deck })),
  list: async () => (await asking.listPresets({})).presets.map(
    (one) => ({ path: one.path, title: one.title }),
  ),
  makes: async (title, folder) => {
    const answer = await asking.createPreset({ title, path: folder })
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
  if (said.load) out.load = ranged(said.load)
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
        goal: goalOf[said.goal] ?? DEFAULTS.goal,
        byDate: said.byDate,
        minutesADay: said.minutesADay,
        newADay: said.newADay,
        reviewsADay: said.reviewsADay,
        retention: said.retention,
        counts: COUNTED[said.counts] ?? DEFAULTS.counts,
        backlog: said.backlog,
        load: said.load,
        hasEvenLoad: said.evenLoad,
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
  evenLoad: settings.hasEvenLoad ?? settings.evenLoad,
  learned: RULING[settings.learned],
  interval: settings.interval,
})

/** A curve as the window carries it. An answer holding none is an empty one. */
const curved = (said: CurveMessage | undefined): Curve => ({
  goal: (said && goalOf[said.goal]) ?? DEFAULTS.goal,
  grid: said?.grid ?? [],
  days: said?.days ?? [],
  at: (said?.at ?? []).map((one) => ({
    reviews: one.reviews,
    minutes: one.minutes,
    retained: one.retained,
    owed: one.owed,
    through: one.through,
    isSufficient: one.enough,
    enough: one.enough,
    closed: one.closed,
    clears: one.clears,
    learned: one.learned,
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
  isValid: true,
  honest: true,
})

const placed = (said: PlaceMessage | undefined): Place =>
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
