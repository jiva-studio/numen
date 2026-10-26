/**
 * Why a preset schedules nothing, as a clause a window reads after something
 * that says which preset it is.
 */
import { StopReason } from '../../gen/numen/flashcards/v1/presets_pb'

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
      return 'no budget in time'
    case StopReason.NO_CARDS:
      return 'no cards a day'
    case StopReason.NO_DAY:
      return 'by no day'
    case StopReason.PAST_DAY:
      return detail.day ? `${detail.day} has passed` : 'the day has passed'
    case StopReason.NO_LOAD:
      return detail.weekday ? `no load on ${detail.weekday}` : 'no load today'
    case StopReason.NO_WEEK:
      return 'no load on any day'
    default:
      return ''
  }
}
