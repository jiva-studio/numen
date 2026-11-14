/**
 * The place of a result in its file as a person reads it: a page of a document, a time or a span of a
 * recording, a chapter of a book.
 */
import type { Words } from '../../words'

/** The fields of a place that are written. */
export interface PlaceFields {
  readonly page?: number
  readonly pageCount?: number
  readonly positionSeconds?: number
  readonly endSeconds?: number
  readonly cfi?: string
  readonly chapter?: string
}

const SECONDS_PER_MINUTE = 60
const SECONDS_PER_HOUR = 3600

const padTwo = (value: number): string => String(value).padStart(2, '0')

/** Writes seconds as m:ss, or h:mm:ss from one hour. */
export function formatSeconds(total: number): string {
  const seconds = Math.floor(Math.max(0, total))
  const hours = Math.floor(seconds / SECONDS_PER_HOUR)
  const minutes = Math.floor((seconds % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE)
  const rest = seconds % SECONDS_PER_MINUTE
  if (hours > 0) return `${hours}:${padTwo(minutes)}:${padTwo(rest)}`
  return `${minutes}:${padTwo(rest)}`
}

/** Writes the place: the page and the page count, the time or the span of a recording, or the chapter. */
export function formatPlace(place: PlaceFields | undefined, words: Words): string {
  if (!place) return ''
  if (place.page !== undefined && place.page > 0) {
    const count =
      place.pageCount !== undefined && place.pageCount > 0 ? ` ${words.of} ${place.pageCount}` : ''
    return `${words.page} ${place.page}${count}`
  }
  if (place.positionSeconds !== undefined) {
    const start = formatSeconds(place.positionSeconds)
    return place.endSeconds === undefined ? start : `${start}–${formatSeconds(place.endSeconds)}`
  }
  return place.chapter ?? ''
}
