/**
 * Curve calculation, caching, and throttling for preset tabs.
 */
import { ref, shallowRef, type Ref } from 'vue'
import { answerGuard, type AnswerGuard } from '@/shared/questions'
import {
  DEFAULTS,
  type Curve,
  type PresetCounts,
  type Presets,
  type Settings,
  type SettingsBounds,
} from '../types'
import { goalValue, nearest } from '../lib/curve'
import { shapeOf } from '../lib/fields'
import { approximate } from '../lib/sketch'
import { WORDS as words } from '../words'

/** Reactive state for preset curve calculations. */
export interface CurveState {
  readonly curve: Ref<Curve>
  readonly material: Ref<PresetCounts | null>
  readonly place: Ref<number>
  readonly waiting: Ref<boolean>
  isDrawing: boolean
  shouldDrawAgain: boolean
  shape: string
  isReal: boolean
  readonly asks: AnswerGuard
  readonly answers: Map<string, Curve>
}

export function createCurveState(
  initialSettings: Settings = DEFAULTS,
  today: string,
  bounds: SettingsBounds,
): CurveState {
  return {
    curve: shallowRef<Curve>(approximate(initialSettings, today, bounds)),
    material: shallowRef<PresetCounts | null>(null),
    place: ref(0),
    waiting: ref(true),
    isDrawing: false,
    shouldDrawAgain: false,
    shape: '',
    isReal: false,
    asks: answerGuard(),
    answers: new Map<string, Curve>(),
  }
}

/** Whether two curves are drawn over the same range, place for place. */
export const isSameGrid = (one: readonly number[], two: readonly number[]): boolean =>
  one.length === two.length && one.every((value, at) => value === two[at])

/** Where the knob stands on a curve: the preset's own place, or the nearest. */
export const getKnobPosition = (curve: Curve, settings: Settings, today: string): number =>
  curve.now.at >= 0 ? curve.now.at : Math.max(nearest(curve.grid, goalValue(settings, today)), 0)

/** Applies a landed curve and updates the material counts. */
export const applyCurveAnswer = (state: CurveState, curve: Curve): void => {
  state.curve.value = curve
  state.material.value = {
    decks: curve.decks,
    cards: curve.cards,
    overdue: curve.overdue,
    unbegun: curve.unbegun,
  }
  state.isReal = true
  state.waiting.value = false
}

/** One curve, asked for and landed. */
const fetchCurve = async (
  state: CurveState,
  path: string,
  getSettings: () => Settings,
  core: Presets,
  bounds: SettingsBounds,
  today: string,
  setErrorMessage: (msg: string) => void,
): Promise<void> => {
  const mine = state.asks.ask()
  const settings = getSettings()
  const shape = shapeOf(settings)
  state.shape = shape
  const riding = state.curve.value.grid

  const standsAlready = state.isReal && state.curve.value.goal === settings.goal
  if (standsAlready) {
    state.curve.value = { ...state.curve.value, honest: false }
  } else {
    const meanwhile = approximate(settings, today, bounds)
    state.curve.value = meanwhile
    state.place.value = Math.max(meanwhile.now.at, 0)
  }
  state.isReal = false
  state.waiting.value = true

  let answer: Curve
  try {
    answer = await core.curve(path, settings)
  } catch {
    if (!mine.current) return
    setErrorMessage(words.noCurve)
    state.waiting.value = false
    return
  }
  if (!mine.current) return
  state.answers.set(shape, answer)
  applyCurveAnswer(state, answer)
  if (standsAlready && isSameGrid(riding, answer.grid)) return
  state.place.value = getKnobPosition(answer, settings, today)
}

/** Calculates the curve for the goal these settings name. */
export const updateCurves = async (
  state: CurveState,
  path: string,
  getSettings: () => Settings,
  core: Presets,
  bounds: SettingsBounds,
  today: string,
  setErrorMessage: (msg: string) => void,
): Promise<void> => {
  const settings = getSettings()
  const shape = shapeOf(settings)
  const heldCurve = state.answers.get(shape)
  if (heldCurve) {
    state.asks.drop()
    state.shape = shape
    applyCurveAnswer(state, heldCurve)
    state.place.value = getKnobPosition(heldCurve, settings, today)
    return
  }

  if (state.isDrawing) {
    state.shouldDrawAgain = true
    return
  }

  state.isDrawing = true
  try {
    await fetchCurve(state, path, getSettings, core, bounds, today, setErrorMessage)
  } finally {
    state.isDrawing = false
  }

  if (!state.shouldDrawAgain) return
  state.shouldDrawAgain = false
  await updateCurves(state, path, getSettings, core, bounds, today, setErrorMessage)
}
