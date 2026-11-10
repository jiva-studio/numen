/**
 * What the bar draws of its slots in the room it has: the widths of the entries as they were last
 * drawn, the room of the start area, and the fit of the two over them.
 */
import { computed, onBeforeUnmount, onMounted, onUpdated, shallowRef, type Ref } from 'vue'
import { useViewport } from '@numen/ui'
import type { SlotItem } from '@numen/plugin-sdk'
import { fitBar } from '../lib/fit'
import { TITLE_MIN_WIDTH } from '../lib/layout'

/** One slot of the bar, and the prefix that makes the keys of its entries unique in the bar. */
export interface BarRow {
  readonly prefix: string
  readonly entries: readonly SlotItem[]
}

/** The key the bar measures and hides an entry of a row by. */
export const getFitKey = (prefix: string, entry: SlotItem): string => `${prefix}#${entry.key}`

const ENTRY = '[data-title-bar-entry]'
const MORE = '[data-title-bar-more]'

function readEntryWidths(
  area: HTMLElement,
  previous: ReadonlyMap<string, number>,
  observer: ResizeObserver | undefined,
): ReadonlyMap<string, number> | null {
  const next = new Map(previous)
  let isChanged = false
  for (const element of area.querySelectorAll<HTMLElement>(ENTRY)) {
    const key = element.getAttribute('data-title-bar-entry') ?? ''
    const width = element.offsetWidth
    if (width > 0 && next.get(key) !== width) {
      next.set(key, width)
      isChanged = true
    }
    observer?.observe(element)
  }
  return isChanged ? next : null
}

export function useBarFit(
  start: Readonly<Ref<HTMLElement | null>>,
  rows: () => readonly BarRow[],
  getReserved: () => number,
) {
  const { viewport } = useViewport(start)
  const widths = shallowRef<ReadonlyMap<string, number>>(new Map())
  const moreWidth = shallowRef(0)
  let observer: ResizeObserver | undefined

  const room = computed(() =>
    viewport.value.width > 0 ? viewport.value.width - getReserved() - TITLE_MIN_WIDTH : null,
  )

  const fit = computed(() => {
    const measured = rows().map(({ prefix, entries }) =>
      entries.map((entry) => {
        const key = getFitKey(prefix, entry)
        return { key, kind: entry.kind, width: widths.value.get(key) ?? 0 }
      }),
    )
    // The menu button is as wide as the button of a command while it is not drawn.
    const buttonWidth =
      measured.flat().find((one) => one.kind === 'command' && one.width > 0)?.width ?? 0
    return fitBar(measured, room.value, moreWidth.value || buttonWidth)
  })

  /** Reads the widths of the entries drawn now; the hidden ones keep the width they had. */
  function measure(): void {
    const area = start.value
    if (!area) return
    const next = readEntryWidths(area, widths.value, observer)
    if (next) widths.value = next
    const more = area.querySelector<HTMLElement>(MORE)?.offsetWidth ?? 0
    if (more > 0 && more !== moreWidth.value) moreWidth.value = more
  }

  onMounted(() => {
    if (typeof ResizeObserver !== 'undefined') observer = new ResizeObserver(() => measure())
    measure()
  })
  onUpdated(measure)
  onBeforeUnmount(() => observer?.disconnect())

  return { fit }
}
