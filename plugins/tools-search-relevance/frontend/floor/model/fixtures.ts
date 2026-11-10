import { vi } from 'vitest'
import { CALIBRATION_OUTCOME, FLOOR_SOURCE } from './codes'
import type { FloorCalibration, FloorClient, FloorState } from './types'

export const CALIBRATION: FloorCalibration = {
  median: 0.27,
  spread: 0.07,
  histogram: [0, 1, 4, 9, 4, 1, 0, 0, 0, 0],
  queryCount: 200,
  passageCount: 500,
  calibratedAt: 1_792_000_000,
}

export function createState(patch: Partial<FloorState> = {}): FloorState {
  return {
    floor: patch.floor ?? 0.41,
    source: patch.source ?? FLOOR_SOURCE.CALIBRATED,
    model: patch.model ?? 'bge-m3',
    automaticFloor: patch.automaticFloor ?? 0.41,
    isManual: patch.isManual ?? false,
    manualFloor: patch.manualFloor ?? 0,
    calibration: 'calibration' in patch ? patch.calibration : CALIBRATION,
  }
}

export interface FakeClient extends FloorClient {
  getFloor: ReturnType<typeof vi.fn<FloorClient['getFloor']>>
  calibrateFloor: ReturnType<typeof vi.fn<FloorClient['calibrateFloor']>>
  setFloor: ReturnType<typeof vi.fn<FloorClient['setFloor']>>
  probeFloor: ReturnType<typeof vi.fn<FloorClient['probeFloor']>>
}

export function createFakeClient(state: FloorState = createState()): FakeClient {
  return {
    getFloor: vi.fn<FloorClient['getFloor']>(async () => state),
    calibrateFloor: vi.fn<FloorClient['calibrateFloor']>(async () => ({
      outcome: CALIBRATION_OUTCOME.DONE,
      state,
    })),
    setFloor: vi.fn<FloorClient['setFloor']>(async (request) => ({
      state: {
        ...state,
        isManual: request.isManual,
        manualFloor: request.floor,
        floor: request.isManual ? request.floor : state.automaticFloor,
      },
    })),
    probeFloor: vi.fn<FloorClient['probeFloor']>(async () => ({
      bestSimilarity: 0.46,
      floor: state.floor,
      doesPass: true,
    })),
  }
}
