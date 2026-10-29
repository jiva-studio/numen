/**
 * Pure projections of a floor state into the words and numbers the panel draws.
 */
import { SLIDER_MAX, SLIDER_MIN, WORDS } from '../words'
import { FLOOR_SOURCE } from './codes'
import type { FloorCalibration, FloorState } from './types'

export function formatFloor(value: number): string {
  return value.toFixed(2)
}

export function getSourceLabel(source: number): string {
  switch (source) {
    case FLOOR_SOURCE.CALIBRATED:
      return WORDS.sourceCalibrated
    case FLOOR_SOURCE.MODEL:
      return WORDS.sourceModel
    case FLOOR_SOURCE.FILE:
      return WORDS.sourceFile
    case FLOOR_SOURCE.MANUAL:
      return WORDS.sourceManual
    default:
      return WORDS.sourceDefault
  }
}

export function formatNoise(calibration: FloorCalibration): string {
  return `median ${formatFloor(calibration.median)} · spread ${formatFloor(calibration.spread)}`
}

function formatDate(seconds: bigint | number): string {
  return new Date(Number(seconds) * 1000).toISOString().slice(0, 10)
}

export function formatCalibrationSummary(calibration: FloorCalibration): string {
  const when = formatDate(calibration.calibratedAt)
  return `${calibration.queryCount} titles × ${calibration.passageCount} passages · calibrated ${when}`
}

/** The line under the noise histogram: what was measured, or why nothing was. */
export function getCalibrationNote(state: FloorState, isTooFew: boolean): string {
  if (state.calibration !== undefined) return formatCalibrationSummary(state.calibration)
  return isTooFew ? WORDS.tooFew : WORDS.pending
}

export interface HistogramBar {
  /** Height as a share of the tallest bin, from 0 to 100. */
  heightPercent: number
  /** Whether the bin lies at or above the threshold. */
  isAbove: boolean
}

/** The bins of a histogram that spans similarities from zero to one. */
export function getHistogramBars(histogram: readonly number[], floor: number): HistogramBar[] {
  const tallest = Math.max(1, ...histogram)
  return histogram.map((count, index) => ({
    heightPercent: (count / tallest) * 100,
    isAbove: (index + 1) / histogram.length > floor,
  }))
}

/** Where a similarity sits along a track from zero to one, as a percentage. */
export function getMarkerPercent(value: number): number {
  return Math.min(100, Math.max(0, value * 100))
}

/** The text alternative of the histogram. */
export function describeHistogram(calibration: FloorCalibration, floor: number): string {
  return `${WORDS.histogramAlt}: ${formatNoise(calibration)}; ${WORDS.histogramMarker.toLowerCase()} ${formatFloor(floor)}`
}

/** The value the slider starts at when the person turns to the manual mode. */
export function getInitialManual(state: FloorState): number {
  const start = state.manualFloor > 0 ? state.manualFloor : state.floor
  return Math.min(SLIDER_MAX, Math.max(SLIDER_MIN, start))
}

/** The sentence under the query check. */
export function describeProbe(bestSimilarity: number, doesPass: boolean): string {
  if (bestSimilarity === 0) return WORDS.probeEmpty
  const verdict = doesPass ? 'above the threshold (passes)' : 'below the threshold'
  return `Best match ${formatFloor(bestSimilarity)} — ${verdict}`
}

export function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
