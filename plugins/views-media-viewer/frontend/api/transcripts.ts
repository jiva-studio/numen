/**
 * The transcript of a recording as the live text version of its file: one cue to a segment, the
 * cue's span in seconds in the segment's selector `t`.
 */
import type { LiveText, VaultTexts } from '@numen/plugin-sdk'
import { getSelectorSpan } from '../lib/place'
import type { Recordings } from '../types'
import { findSegmentTime, getCues, getSegments, removeTrailingLineBreaks } from './segments'

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
