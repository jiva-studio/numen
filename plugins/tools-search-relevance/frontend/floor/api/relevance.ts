/**
 * The JSON messages of the relevance service the plugin's backend serves. Keys and meaning match
 * the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */

/** The service name the manifest declares. */
export const SERVICE = 'md.numen.tools.search.relevance'

/** Where the floor in force comes from. */
export type FloorSource = 'manual' | 'calibrated' | 'model' | 'default'

/** How a calibration ended when it did not fail. */
export type CalibrationOutcome = 'calibrated' | 'too-few'

type GetFloorRequest = Record<string, never>

export interface Calibration {
  readonly median: number
  readonly spread: number
  readonly histogram: readonly number[]
  readonly queryCount: number
  readonly passageCount: number
  readonly calibratedAt: number
}

export interface GetFloorResponse {
  readonly floor: number
  readonly source: FloorSource
  readonly model: string
  readonly automaticFloor: number
  readonly isManual: boolean
  readonly manualFloor: number
  readonly calibration?: Calibration | undefined
}

type CalibrateFloorRequest = Record<string, never>

interface CalibrateFloorResponse {
  readonly outcome: CalibrationOutcome
  readonly state: GetFloorResponse
}

interface SetFloorRequest {
  readonly isManual: boolean
  readonly floor: number
}

interface SetFloorResponse {
  readonly state: GetFloorResponse
}

interface ProbeFloorRequest {
  readonly query: string
}

export interface ProbeFloorResponse {
  readonly bestSimilarity: number
  readonly floor: number
  readonly doesPass: boolean
}

/** What a calibration answered. A state the answer leaves out is read again. */
interface FloorCalibrationResult {
  outcome: CalibrationOutcome
  state?: GetFloorResponse | undefined
}

/** What a choice of the floor answered. A state the answer leaves out is read again. */
interface FloorWriteResult {
  state?: GetFloorResponse | undefined
}

/** The calls of the relevance service, as the panel makes them. */
export interface FloorClient {
  getFloor(request: GetFloorRequest): Promise<GetFloorResponse>
  calibrateFloor(request: CalibrateFloorRequest): Promise<FloorCalibrationResult>
  setFloor(request: SetFloorRequest): Promise<FloorWriteResult>
  probeFloor(request: ProbeFloorRequest): Promise<ProbeFloorResponse>
}

/** The request and response of each method by name. */
export interface Methods {
  GetFloor: { request: GetFloorRequest; response: GetFloorResponse }
  CalibrateFloor: { request: CalibrateFloorRequest; response: CalibrateFloorResponse }
  SetFloor: { request: SetFloorRequest; response: SetFloorResponse }
  ProbeFloor: { request: ProbeFloorRequest; response: ProbeFloorResponse }
}
