/**
 * Position of each run of the rendered document, and which runs are marked.
 *
 * The runs are read from the text itself, so they are gathered again whenever
 * the columns are laid out again. Marked ranges are expressed as byte offsets of
 * the book's text, and the marks map those to positions on the page.
 */
import { onBeforeUnmount, shallowRef, watch } from 'vue'
import type { ShallowRef } from 'vue'

import { HIGHLIGHT, OTHER_HIGHLIGHT, highlight, unhighlight } from '../lib/highlight'
import { createMarks, findOffset, rangesOver, getRuns, type Run } from '../lib/runs'
import type { Mark } from '../lib/spread'
import type { SettledBookProps } from '../lib/props'

/**
 * `getEdge` is the one measurement of the browser this makes: the near edge of
 * the columns, which every mark is read against.
 */
export function useBookMarks(
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  getEdge: (of: HTMLElement) => number,
) {
  /** Position of each run of the text. */
  const marks = shallowRef<readonly Mark[]>([])

  /** Each run of the text, in the order the document sets them. */
  let runs: readonly Run[] = []

  /** The runs of the drawn document, and where the columns put each of them. */
  const gather = () => {
    const text = paper.value
    if (!text) return

    // the area they are carried across: a turn under way carries both, and the
    // one measured against the other is where the run will come to rest.
    runs = getRuns(text)
    marks.value = createMarks(runs, getEdge(text))
  }

  const findFragmentOffset = (fragment: string): number | undefined => {
    const text = paper.value
    return text ? findOffset(text, runs, fragment) : undefined
  }

  /** Highlights the runs in `highlights`, and the runs in `otherHighlights` more faintly. */
  const markRuns = () => {
    highlight(HIGHLIGHT, props, rangesOver(runs, props.highlights))
    highlight(OTHER_HIGHLIGHT, props, rangesOver(runs, props.otherHighlights))
  }

  watch([() => props.highlights, () => props.otherHighlights], markRuns)

  onBeforeUnmount(() => {
    unhighlight(props)
  })

  return { marks, gather, findFragmentOffset, markRuns }
}
