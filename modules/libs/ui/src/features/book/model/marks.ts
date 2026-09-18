/**
 * Where each run of the drawn document stands, and which of them are marked.
 *
 * The runs are read off the text itself, so they are gathered again whenever the
 * columns are set again. What is marked is asked for in bytes of the book's
 * text, and the marks are what turn that into a place on the page.
 */
import { onBeforeUnmount, shallowRef, watch } from 'vue'
import type { ShallowRef } from 'vue'

import { HIGHLIGHT, OTHER_HIGHLIGHT, highlight, unhighlight } from '../lib/highlight'
import { marksIn, offsetAt, rangesOver, getRunsIn, type Run } from '../lib/runs'
import type { Mark } from '../lib/spread'
import type { SettledBookProps } from '../lib/props'

/**
 * `edgeOf` is the one measurement of the browser this makes: the near edge of
 * the columns, which every mark is read against.
 */
export function useBookMarks(
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  edgeOf: (of: HTMLElement) => number,
) {
  /** Where each run of the text stands. */
  const marks = shallowRef<readonly Mark[]>([])

  /** Each run of the text, in the order the document sets them. */
  let runs: readonly Run[] = []

  /** The runs of the drawn document, and where the columns put each of them. */
  const gather = () => {
    const text = paper.value
    if (!text) return

    // Where a run stands is read against the columns it stands in and not against
    // the area they are carried across: a turn under way carries both, and the
    // one measured against the other is where the run will come to rest.
    runs = getRunsIn(text)
    marks.value = marksIn(runs, edgeOf(text))
  }

  /** The offset a place named inside the drawn document stands at. */
  const placeAt = (fragment: string): number | undefined => {
    const text = paper.value
    return text ? offsetAt(text, runs, fragment) : undefined
  }

  /** The runs asked about marked where they stand, and the rest more faintly. */
  const markRuns = () => {
    highlight(HIGHLIGHT, props, rangesOver(runs, props.highlights))
    highlight(OTHER_HIGHLIGHT, props, rangesOver(runs, props.otherHighlights))
  }

  watch([() => props.highlights, () => props.otherHighlights], markRuns)

  onBeforeUnmount(() => {
    unhighlight(props)
  })

  return { marks, gather, placeAt, markRuns }
}
