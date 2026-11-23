/**
 * Curve calculation, caching, and throttling for preset tabs.
 */
import type { Curve, Presets, Settings, SettingsBounds } from '../../lib/presets'
import { getKnobPosition, isSameGrid } from '../lib/curveGrid'
import { getShapeKey } from '../lib/fields'
import { estimateCurve } from '../lib/sketch'
import { WORDS as words } from '../lib/words'
import { applyCurveAnswer, type CurveState } from './curveState'

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
    answer = await core.getCurve(path, settings)
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
