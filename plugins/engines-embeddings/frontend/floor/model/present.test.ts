import { describe, expect, it } from 'vitest'
import { FLOOR_SOURCE } from './codes'
import { CALIBRATION, createState } from './fixtures'
import {
  describeProbe,
  formatCalibrationSummary,
  formatNoise,
  getCalibrationNote,
  getErrorMessage,
  getHistogramBars,
  getInitialManual,
  getMarkerPercent,
  getSourceLabel,
} from './present'

describe('present', () => {
  it('names each source', () => {
    expect(getSourceLabel(FLOOR_SOURCE.CALIBRATED)).toBe('Calibrated on your vault')
    expect(getSourceLabel(FLOOR_SOURCE.MODEL)).toBe('Model profile')
    expect(getSourceLabel(FLOOR_SOURCE.DEFAULT)).toBe('Default')
    expect(getSourceLabel(FLOOR_SOURCE.FILE)).toBe('Settings file')
    expect(getSourceLabel(FLOOR_SOURCE.UNSPECIFIED)).toBe('Default')
  })

  it('formats the noise and the summary', () => {
    expect(formatNoise(CALIBRATION)).toBe('median 0.27 · spread 0.07')
    expect(formatCalibrationSummary(CALIBRATION)).toBe(
      '200 titles × 500 passages · calibrated 2026-10-14',
    )
  })

  it('chooses the note under the histogram', () => {
    expect(getCalibrationNote(createState(), false)).toContain('200 titles')
    expect(getCalibrationNote(createState({ calibration: undefined }), true)).toBe(
      'Not enough notes yet (needs ≥ 20)',
    )
    expect(getCalibrationNote(createState({ calibration: undefined }), false)).toBe(
      'Calibration pending',
    )
  })

  it('scales the bars to the tallest bin and splits them at the threshold', () => {
    const bars = getHistogramBars([1, 4, 2, 0], 0.5)
    expect(bars.map((bar) => bar.heightPercent)).toEqual([25, 100, 50, 0])
    expect(bars.map((bar) => bar.isAbove)).toEqual([false, false, true, true])
  })

  it('draws an all-zero histogram flat', () => {
    expect(getHistogramBars([0, 0], 0.5).map((bar) => bar.heightPercent)).toEqual([0, 0])
  })

  it('keeps the marker on the track', () => {
    expect(getMarkerPercent(0.41)).toBeCloseTo(41)
    expect(getMarkerPercent(-1)).toBe(0)
    expect(getMarkerPercent(2)).toBe(100)
  })

  it('starts the manual value from the saved one or the floor in force, inside the slider', () => {
    expect(getInitialManual(createState({ manualFloor: 0.6 }))).toBe(0.6)
    expect(getInitialManual(createState({ manualFloor: 0, floor: 0.41 }))).toBe(0.41)
    expect(getInitialManual(createState({ manualFloor: 0, floor: 0.99 }))).toBe(0.95)
    expect(getInitialManual(createState({ manualFloor: 0, floor: 0 }))).toBe(0.05)
  })

  it('describes the probe', () => {
    expect(describeProbe(0.46, true)).toBe('Best match 0.46 — above the threshold (passes)')
    expect(describeProbe(0.3, false)).toBe('Best match 0.30 — below the threshold')
    expect(describeProbe(0, false)).toBe('The vault has no indexed passages yet.')
  })

  it('reads the message of any thrown value', () => {
    expect(getErrorMessage(new Error('x'))).toBe('x')
    expect(getErrorMessage('y')).toBe('y')
  })
})
