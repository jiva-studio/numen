/**
 * Curve calculation, caching, and throttling for preset tabs.
 */
import { ref, shallowRef, type Ref } from 'vue'
import { createRequestGuard, type RequestGuard } from '@numen/ui'
import {
  DEFAULTS,
  type Curve,
  type PresetCounts,
  type Presets,
  type Settings,
  type SettingsBounds,
} from '../../lib/presets'
import { getGoalValue, findNearestGridValue } from '../lib/curve'
import { getShapeKey } from '../lib/fields'
import { estimateCurve } from '../lib/sketch'
import { WORDS as words } from '../lib/words'

/** Reactive state for preset curve calculations. */
export interface CurveState {
  readonly current: {
    readonly curve: Ref<Curve>
    readonly material: Ref<PresetCounts | null>
    readonly place: Ref<number>
    readonly isWaiting: Ref<boolean>
  }
  readonly drawing: {
    isDrawing: boolean
    shouldDrawAgain: boolean
    shape: string
    isReal: boolean
  }
  readonly requests: {
    readonly asks: RequestGuard
    readonly answers: Map<string, Curve>
  }
}

export function createCurveState(
  initialSettings: Settings = DEFAULTS,
  today: string,
  bounds: SettingsBounds,
): CurveState {
  return {
    current: {
      curve: shallowRef<Curve>(estimateCurve(initialSettings, today, bounds)),
      material: shallowRef<PresetCounts | null>(null),
      place: ref(0),
      isWaiting: ref(true),
    },
    drawing: { isDrawing: false, shouldDrawAgain: false, shape: '', isReal: false },
    requests: { asks: createRequestGuard(), answers: new Map<string, Curve>() },
  }
}

/** Whether two curves are drawn over the same range, place for place. */
export const isSameGrid = (one: readonly number[], two: readonly number[]): boolean =>
  one.length === two.length && one.every((value, index) => value === two[index])

/** Where the knob relies on a curve: the preset's own place, or the nearest. */
export const getKnobPosition = (curve: Curve, settings: Settings, today: string): number =>
  curve.now.index >= 0
    ? curve.now.index
    : Math.max(findNearestGridValue(curve.grid, getGoalValue(settings, today)), 0)

/** Applies a landed curve and updates the material counts. */
export const applyCurveAnswer = (state: CurveState, curve: Curve): void => {
  state.current.curve.value = curve
  state.current.material.value = {
    decks: curve.decks,
    cards: curve.cards,
    overdue: curve.overdue,
    unbegun: curve.unbegun,
  }
  state.drawing.isReal = true
  state.current.isWaiting.value = false
}

export interface CurveContext {
  path: string
  getSettings: () => Settings
  core: Presets
  bounds: SettingsBounds
  today: string
  setErrorMessage: (msg: string) => void
}

const fetchCurve = async (state: CurveState, context: CurveContext): Promise<void> => {
  const { path, getSettings, core, bounds, today, setErrorMessage } = context
  const mine = state.requests.asks.acquireRequest()
  const settings = getSettings()
  const shape = getShapeKey(settings)
  state.drawing.shape = shape
  const riding = state.current.curve.value.grid

  const isCurrent = state.drawing.isReal && state.current.curve.value.goal === settings.goal
  if (isCurrent) {
    state.current.curve.value = { ...state.current.curve.value, isHonest: false }
  } else {
    const meanwhile = estimateCurve(settings, today, bounds)
    state.current.curve.value = meanwhile
    state.current.place.value = Math.max(meanwhile.now.index, 0)
  }
  state.drawing.isReal = false
  state.current.isWaiting.value = true

  let answer: Curve
  try {
    answer = await core.curve(path, settings)
  } catch {
    // The window states what failed; the error returned by the call adds nothing the user can act on.
    if (!mine.isCurrent) return
    setErrorMessage(words.noCurve)
    state.current.isWaiting.value = false
    return
  }
  if (!mine.isCurrent) return
  state.requests.answers.set(shape, answer)
  applyCurveAnswer(state, answer)
  if (isCurrent && isSameGrid(riding, answer.grid)) return
  state.current.place.value = getKnobPosition(answer, settings, today)
}

/** Calculates the curve for the goal these settings name. */
export const updateCurves = async (state: CurveState, context: CurveContext): Promise<void> => {
  const { getSettings, today } = context
  const settings = getSettings()
  const shape = getShapeKey(settings)
  const cachedCurve = state.requests.answers.get(shape)
  if (cachedCurve) {
    state.requests.asks.cancel()
    state.drawing.shape = shape
    applyCurveAnswer(state, cachedCurve)
    state.current.place.value = getKnobPosition(cachedCurve, settings, today)
    return
  }

  if (state.drawing.isDrawing) {
    state.drawing.shouldDrawAgain = true
    return
  }

  state.drawing.isDrawing = true
  try {
    await fetchCurve(state, context)
  } finally {
    state.drawing.isDrawing = false
  }

  if (!state.drawing.shouldDrawAgain) return
  state.drawing.shouldDrawAgain = false
  await updateCurves(state, context)
}
