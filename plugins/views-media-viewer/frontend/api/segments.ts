import type { TextSegment } from '@numen/plugin-sdk'
import { getSelectorSpan } from '../lib/place'
import type { TextRange, Cue } from '../types'

/** The cues of a live text: its segments that name a span of the recording. */
export const getCues = (segments: readonly TextSegment[]): readonly Cue[] =>
  segments.flatMap((segment) => {
    const t = getSelectorSpan(segment.selector)
    if (t === undefined) return []
    return [{ text: removeTrailingLineBreaks(segment.text), from: toMS(t[0]), to: toMS(t[1]) }]
  })

export const removeTrailingLineBreaks = (text: string): string => {
  let end = text.length
  while (end > 0 && text[end - 1] === '\n') end--
  return text.slice(0, end)
}

/** The cues as segments of a text version, one line to a cue. */
export const getSegments = (cues: readonly Cue[]): readonly TextSegment[] =>
  cues.map((cue) => ({ text: `${cue.text}\n`, selector: { t: [cue.from / 1000, cue.to / 1000] } }))

/**
 * The millisecond the segment at a span's start begins, and null where no segment holds it. The
 * span counts UTF-16 code units of the joined text, as a JavaScript string does.
 */
export const findSegmentTime = (
  segments: readonly TextSegment[],
  span: TextRange,
): number | null => {
  let offset = 0
  for (const segment of segments) {
    const end = offset + segment.text.length
    if (span.start < end) {
      const t = getSelectorSpan(segment.selector)
      return t === undefined ? null : toMS(t[0])
    }
    offset = end
  }
  return null
}

const toMS = (seconds: number): number => Math.round(seconds * 1000)
