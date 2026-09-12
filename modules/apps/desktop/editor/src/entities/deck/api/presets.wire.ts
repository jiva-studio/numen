/** The presets of a vault, as the window asks for them and as the schema writes them. */
import {
  BudgetUnit as BudgetUnits,
  Rule as Rules,
  type Bounds as BoundsMessage,
  type Curve as CurveMessage,
  type ErrorCode as ProtoErrorCode,
  type Place as PlaceMessage,
  type Preset as PresetMessage,
  type Settings as SettingsMessage,
  type SettingsBounds as SettingsBoundsMessage,
} from '@numen/protocol'
import { goalNames, goalOf, namesOf } from '@numen/wire'
import { fingerprint, errorIn, staleIn, stamp } from '@/shared/answers'
import { presetsService } from '@/shared/clients'
import { DEFAULTS, NOWHERE } from '../lib/presets'
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
} from '../lib/presets'

/** The same questions, in the shape the window asks them. */
export const presets: Presets = {
  read: async (path) => parseRead(await presetsService.readPreset({ path })),
  getDeckPreset: async (deck) => parseRead(await presetsService.getDeckPreset({ deck })),
  list: async () =>
    (await presetsService.listPresets({})).presets.map((one) => ({ path: one.path, title: one.title })),
  createPreset: async (title, folder) => {
    const answer = await presetsService.createPreset({ title, path: folder })
    return { path: answer.path, error: errorIn(answer) }
  },
  scheduleDeck: async (deck, preset, seen) => {
    const answer = await presetsService.scheduleDeck({
      deck,
      preset,
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { error: errorIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  write: async (path, settings, seen) => {
    const answer = await presetsService.writePreset({
      path,
      settings: toSettingsMessage(settings),
      ...(seen === '' ? {} : { seen: fingerprint(seen) }),
    })
    return { error: errorIn(answer), changed: staleIn(answer), at: stamp(answer.at) ?? '' }
  },
  curve: async (path, settings) => {
    const answer = await presetsService.computeCurve({ path, settings: toSettingsMessage(settings) })
    return parseCurve(answer.curve)
  },
}

/** What a read answered, whichever of the two asked it. */
const parseRead = (answer: {
  preset?: PresetMessage | undefined
  error?: ProtoErrorCode | undefined
  at?: { path: string; size: bigint; mtime: bigint } | undefined
  bounds?: SettingsBoundsMessage | undefined
}): ReadResult => ({
  preset: answer.preset ? parsePreset(answer.preset) : null,
  error: errorIn(answer),
  at: stamp(answer.at) ?? '',
  bounds: parseSettingsBounds(answer.bounds),
})

/** How far each setting goes, as the read answered it. */
const parseSettingsBounds = (all: SettingsBoundsMessage | undefined): SettingsBounds => {
  const out: { -readonly [field in keyof SettingsBounds]: Bounds } = {}
  if (all === undefined) return out
  if (all.minutesADay) out.minutesADay = parseBounds(all.minutesADay)
  if (all.newADay) out.newADay = parseBounds(all.newADay)
  if (all.reviewsADay) out.reviewsADay = parseBounds(all.reviewsADay)
  if (all.retention) out.retention = parseBounds(all.retention)
  if (all.backlog) out.backlog = parseBounds(all.backlog)
  if (all.interval) out.interval = parseBounds(all.interval)
  if (all.load) out.load = parseBounds(all.load)
  return out
}

/** One pair of ends, in the window's own words. */
const parseBounds = (bounds: BoundsMessage): Bounds => ({
  least: bounds.least,
  most: bounds.most,
})

/** One preset as the window carries it. */
const parsePreset = (one: PresetMessage): Preset => ({
  path: one.path,
  title: one.title,
  settings: settingsOf(one.settings),
  problems: one.problems,
  stops: one.stops,
  stopsOn: one.stopsOn,
})

/** The settings in the window's own words. A preset carrying none is the defaults. */
const settingsOf = (settings: SettingsMessage | undefined): Settings =>
  settings === undefined
    ? DEFAULTS
    : {
        goal: goalOf[settings.goal] ?? DEFAULTS.goal,
        byDate: settings.byDate,
        minutesADay: settings.minutesADay,
        newADay: settings.newADay,
        reviewsADay: settings.reviewsADay,
        retention: settings.retention,
        counts: COUNTED[settings.counts] ?? DEFAULTS.counts,
        backlog: settings.backlog,
        load: settings.load,
        evenLoad: settings.evenLoad,
        learned: LEARNED[settings.learned] ?? DEFAULTS.learned,
        interval: settings.interval,
      }

/** The settings in the shape the schema carries them. */
const toSettingsMessage = (settings: Settings) => ({
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
const parseCurve = (curve: CurveMessage | undefined): Curve => ({
  goal: (curve && goalOf[curve.goal]) ?? DEFAULTS.goal,
  grid: curve?.grid ?? [],
  days: curve?.days ?? [],
  at: (curve?.at ?? []).map((one) => ({
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
  now: parsePlace(curve?.now),
  suggested: parsePlace(curve?.suggested),
  decks: curve?.decks ?? 0,
  cards: curve?.cards ?? 0,
  overdue: curve?.overdue ?? 0,
  unbegun: curve?.unbegun ?? 0,
  isValid: true,
  honest: true,
})

const parsePlace = (place: PlaceMessage | undefined): Place =>
  place === undefined ? NOWHERE : { at: place.at, value: place.value, day: place.day }

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
