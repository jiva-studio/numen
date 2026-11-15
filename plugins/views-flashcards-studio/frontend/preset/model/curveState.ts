import { ref, shallowRef, type Ref } from 'vue'
import { createRequestGuard, type RequestGuard } from '@numen/ui'
import {
  DEFAULTS,
  type Curve,
  type PresetCounts,
  type Settings,
  type SettingsBounds,
} from '../../lib/presets'
import { estimateCurve } from '../lib/sketch'

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
