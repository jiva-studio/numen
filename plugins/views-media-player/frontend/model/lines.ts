/**
 * The transcript as the editor lays it out: one line per cue, each paired with the
 * milliseconds it covers and its clock time.
 */
import { computed, type Ref } from 'vue'
import { clock } from '@numen/ui'
import { spanCues, type Cue } from '../lib/cues'

export function useTranscriptLines(cues: Ref<readonly Cue[]>, prose: Ref<string>) {
  /**
   * The lines on screen with the milliseconds they cover. The lines are what the user
   * edits, and these follow them until the file is written.
   *
   * A recording with no heard speech and no typed text has no lines, and the tab
   * shows that state in their place.
   */
  const spans = computed(() =>
    cues.value.length === 0 && prose.value === '' ? [] : spanCues(cues.value, prose.value),
  )

  /**
   * Whether the text carries the time of each span. A page is prose and carries
   * none: there is nothing to seek and no current line.
   */
  const timed = computed(() => cues.value.length > 0)

  /** Whether anything at all has been fetched or heard here. */
  const hasText = computed(() => timed.value || prose.value !== '')

  /**
   * The clock time of each line, as drawn in the editor's gutter. These follow the
   * words alone, so a transcript of any length is written out once and unchanged
   * while the recording plays.
   */
  const times = computed(() => spans.value.map((cue) => clock(cue.from)))

  /** How far into the recording the words written down reach, in milliseconds. */
  const transcribedDuration = computed(() => cues.value.at(-1)?.to ?? 0)

  return { spans, timed, hasText, times, transcribedDuration }
}
