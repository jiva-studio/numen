import type { FloorCalibration } from './types'

function formatDate(seconds: number): string {
  return new Date(Number(seconds) * 1000).toISOString().slice(0, 10)
}

export function formatCalibrationSummary(calibration: FloorCalibration): string {
  const when = formatDate(calibration.calibratedAt)
  return `${calibration.queryCount} titles × ${calibration.chunkCount} passages · calibrated ${when}`
}
