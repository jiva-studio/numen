import { describe, expect, it } from 'vitest'
import { StopReason } from '../../api/types'

import { formatStopReasonMessage } from './stop'

const SILENT: StopReason[] = [StopReason.UNSPECIFIED, StopReason.NOTHING]

describe('a stop reason', () => {
  it('says nothing for a preset that schedules', () => {
    for (const reason of SILENT) expect(formatStopReasonMessage(reason)).toBe('')
  })

  it('has a clause of its own for every reason that stops', () => {
    const values = Object.values(StopReason)
      .filter((reason) => !SILENT.includes(reason))
      .map((reason) => formatStopReasonMessage(reason))
    expect(values.every((one) => one !== '')).toBe(true)
    expect(new Set(values).size).toBe(values.length)
  })

  it('names the day a preset aimed at', () => {
    expect(formatStopReasonMessage(StopReason.PAST_DAY, { day: '3 March' })).toBe(
      'target date 3 March passed',
    )
    expect(formatStopReasonMessage(StopReason.PAST_DAY)).toBe('target date passed')
  })

  it('names the weekday that carries no load', () => {
    expect(formatStopReasonMessage(StopReason.NO_LOAD, { weekday: 'Saturday' })).toBe(
      'no load on Saturday',
    )
    expect(formatStopReasonMessage(StopReason.NO_LOAD)).toBe('no load today')
  })
})
