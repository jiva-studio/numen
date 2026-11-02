/** The names the panel reads in the answers of the relevance service. */
import type { CalibrationOutcome, FloorSource } from '../api/relevance'

export const FLOOR_SOURCE = {
  MANUAL: 'manual',
  CALIBRATED: 'calibrated',
  MODEL: 'model',
  DEFAULT: 'default',
} as const satisfies Record<string, FloorSource>

export const CALIBRATION_OUTCOME = {
  DONE: 'calibrated',
  TOO_FEW: 'too-few',
} as const satisfies Record<string, CalibrationOutcome>
