/**
 * The layout of a book drawn in columns: the columns set against the room they
 * are read in, and the spread in front carried to the edge of it.
 *
 * What the columns are is `columns.ts` and where the reader stands is
 * `turns.ts`. What is here is the settling that joins them — the text set
 * again, with one offset kept in front — and what asks for it.
 */
import { onMounted, watch } from 'vue'
import type { ShallowRef } from 'vue'

import { onNextFrame } from '@/shared/lib/clock'
import { useBookColumns } from './columns'
import { useBookTurns } from './turns'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/**
 * The room one document of a book is read in, and everything measured from it.
 * `onMove` is what the reader says when the offset in front changes, and
 * `takeLed` hands over the place a link led to, once.
 */
export function useBookLayout(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  onMove: (at: number) => void,
  takeLed: () => BookLink | undefined,
  edgeOf: (of: HTMLElement) => number,
) {
  const { textSize, viewport, isMeasured, flow, spreadCount, setting, marks, ...column } =
    useBookColumns(area, paper, props, edgeOf)

  const { front, leftInChapter, stand, getKeptOffset, getSpreadOf, turn, standing } = useBookTurns(
    paper,
    props,
    flow,
    spreadCount,
    marks,
    onMove,
  )

  /**
   * The text set in columns again, with one offset kept in front. The columns are
   * measured after the browser has laid them out, and only an area that overflows
   * has a length to measure.
   *
   * A place a link led to is found here, in markup that has only now been drawn.
   */
  const settle = (keep: number, led?: BookLink) => {
    column.measure()
    if (!isMeasured.value) return
    onNextFrame(() => {
      const box = area.value
      const text = paper.value
      if (!box || !text) return
      column.readColumns(box, text)
      const landed =
        led && (led.path === '' || led.path === props.path)
          ? column.placeAt(led.fragment)
          : undefined
      stand(getSpreadOf(landed ?? keep), 'auto')
      if (landed !== undefined) onMove(landed)
      column.markRuns()
    })
  }

  // A reading area of another size, or a text of another size, is another set of
  // columns.
  watch([() => viewport.value.width, () => viewport.value.height, textSize], () => {
    settle(getKeptOffset())
  })

  // Another document is opened at the offset asked for, and there is nothing to
  // keep in front.
  watch(
    () => props.markup,
    () => {
      const place = takeLed()
      standing.value = 0
      settle(props.at, place)
    },
  )

  // An offset asked for from outside is turned to. One reached by the hand is
  // already in front.
  watch(
    () => props.at,
    (at) => {
      if (at < props.span.from || at >= props.span.to) return
      const want = getSpreadOf(at)
      if (want !== standing.value) stand(want, 'smooth')
    },
  )

  onMounted(() => {
    settle(props.at)
    // The columns are counted over the type the book is set in, which arrives
    // after the markup does.
    void document.fonts?.ready.then(() => settle(getKeptOffset()))
  })

  return {
    textSize,
    isMeasured,
    flow,
    spreadCount,
    front,
    leftInChapter,
    setting,
    getKeptOffset,
    settle,
    placeAt: column.placeAt,
    turn,
  }
}
