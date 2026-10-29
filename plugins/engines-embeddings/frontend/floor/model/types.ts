/**
 * The messages of the floor calls, as the panel reads them. The core search client satisfies
 * FloorClient.
 */
export interface FloorCalibration {
  median: number
  spread: number
  histogram: number[]
  queryCount: number
  passageCount: number
  /** Seconds since the epoch. */
  calibratedAt: bigint | number
}

export interface FloorState {
  floor: number
  source: number
  model: string
  automaticFloor: number
  isManual: boolean
  manualFloor: number
  calibration?: FloorCalibration | undefined
}

export interface FloorCalibrationResult {
  outcome: number
  state?: FloorState | undefined
}

export interface FloorWriteResult {
  state?: FloorState | undefined
}

export interface FloorProbeResult {
  bestSimilarity: number
  floor: number
  doesPass: boolean
}

export interface FloorClient {
  getFloor(request: Record<string, never>): Promise<FloorState>
  calibrateFloor(request: Record<string, never>): Promise<FloorCalibrationResult>
  setFloor(request: { isManual: boolean; floor: number }): Promise<FloorWriteResult>
  probeFloor(request: { query: string }): Promise<FloorProbeResult>
}

/** Pauses for a number of milliseconds. */
export type Wait = (ms: number) => Promise<unknown>
