/**
 * The transcript as the editor lays it out: one line to a cue, each against
 * the milliseconds it covers and the moment on a clock it was said at.
 */
import { computed, type Ref } from 'vue'
import { clock } from '@numen/ui'
import { spanCues, type Cue } from '../lib/cues'

export function useTranscriptLines(cues: Ref<readonly Cue[]>, prose: Ref<string>) {
  /**
   * The lines on screen against the milliseconds they cover. The lines are
   * what a person edits, and these follow them until the file is written.
   *
   * A recording nothing was heard in and nothing was typed into has no lines
   * at all, and the tab says so where they would stand.
   */
  const spans = computed(() =>
    cues.value.length === 0 && prose.value === '' ? [] : spanCues(cues.value, prose.value),
  )

  /**
   * Whether the text carries the times each span of it was said at. A page
   * is prose and carries none: there is nothing to seek and no line being said.
   */
  const timed = computed(() => cues.value.length > 0)

  /** Whether anything at all has been fetched or heard here. */
  const written = computed(() => timed.value || prose.value !== '')

  /**
   * The moment each line was said at, on a clock, as the editor's gutter draws
   * them. These follow the words alone, so a transcript of any length is
   * written out once and left alone while the recording plays.
   */
  const times = computed(() => spans.value.map((cue) => clock(cue.from)))

  /** How far into the recording the words written down reach, in milliseconds. */
  const transcribedDuration = computed(() => cues.value.at(-1)?.to ?? 0)

  return { spans, timed, written, times, transcribedDuration }
}
