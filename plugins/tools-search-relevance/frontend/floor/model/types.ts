/**
 * The messages of the floor calls, as the panel reads them. The wire client of the relevance service
 * satisfies FloorClient.
 */
import type { Calibration, GetFloorResponse, ProbeFloorResponse } from '../api/relevance'

export type { FloorClient } from '../api/relevance'

export type FloorCalibration = Calibration
export type FloorState = GetFloorResponse
export type FloorProbeResult = ProbeFloorResponse

/** Pauses for a number of milliseconds. */
export type Wait = (ms: number) => Promise<void>
