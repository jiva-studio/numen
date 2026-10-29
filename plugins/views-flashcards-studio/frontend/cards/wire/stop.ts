/**
 * Why a preset schedules nothing, as a clause to display after text that
 * identifies the preset.
 */
import { StopReason } from '../../api/types'

/** The day and the weekday as a window words them; a reason that names a day reads them. */
export interface StopDetail {
  /** The day the preset aimed at. */
  readonly day?: string | undefined
  /** The day of the week that carries no load. */
  readonly weekday?: string | undefined
}

/** The clause for a reason, and empty for a preset that schedules something. */
export const formatStopReasonMessage = (reason: StopReason, detail: StopDetail = {}): string => {
  switch (reason) {
    case StopReason.NO_MINUTES:
      return 'no time budget'
    case StopReason.NO_CARDS:
      return 'no cards per day'
    case StopReason.NO_DAY:
      return 'no target date'
    case StopReason.PAST_DAY:
      return detail.day ? `target date ${detail.day} passed` : 'target date passed'
    case StopReason.NO_LOAD:
      return detail.weekday ? `no load on ${detail.weekday}` : 'no load today'
    case StopReason.NO_WEEK:
      return 'no load on any day'
    default:
      return ''
  }
}
