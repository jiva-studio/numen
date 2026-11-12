/**
 * The transcript of a recording as the live text version of its file: one cue to a segment, the
 * cue's span in seconds in the segment's selector `t`.
 */
import type { LiveText, TextSegment, VaultTexts } from '@numen/plugin-sdk'
import { getSelectorSpan } from '../lib/place'
import type { Recordings, TextRange, Cue } from '../types'

/** The cues of a live text: its segments that name a span of the recording. */
export const getCues = (segments: readonly TextSegment[]): readonly Cue[] =>
  segments.flatMap((segment) => {
    const t = getSelectorSpan(segment.selector)
    if (t === undefined) return []
    return [{ text: removeTrailingLineBreaks(segment.text), from: toMS(t[0]), to: toMS(t[1]) }]
  })

const removeTrailingLineBreaks = (text: string): string => {
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

const isTimed = (live: LiveText): boolean =>
  live.segments.some((segment) => getSelectorSpan(segment.selector) !== undefined)

/** The transcript calls of a recording tab, over the texts the plugin's context gives. */
export const createTranscripts = (
  getTexts: () => VaultTexts | undefined,
): Pick<Recordings, 'readTranscript' | 'readArticle' | 'writeTranscript' | 'findCueTime'> => {
  const readLiveText = async (path: string): Promise<LiveText | null> => {
    const texts = getTexts()
    if (texts === undefined) return null
    const live = await texts.readLiveText(path)
    return live !== null && isTimed(live) ? live : null
  }
  return {
    readTranscript: async (path) => {
      const live = await readLiveText(path)
      return live === null ? [] : getCues(live.segments)
    },
    readArticle: async (path) => {
      const live = await getTexts()?.readLiveText(path)
      if (!live || isTimed(live)) return ''
      return live.segments.map((segment) => removeTrailingLineBreaks(segment.text)).join('\n')
    },
    writeTranscript: async (path, cues) => {
      const texts = getTexts()
      const live = await readLiveText(path)
      if (texts === undefined || live === null) {
        throw new Error(`${path} has no transcript to edit`)
      }
      const edit = await texts.createEdit(path, live.version.id, getSegments(cues))
      if (!edit.ok) throw new Error(`the edit of ${path} was not saved: ${edit.error}`)
    },
    findCueTime: async (path, span) => {
      const live = await readLiveText(path)
      return live === null ? null : findSegmentTime(live.segments, span)
    },
  }
}
