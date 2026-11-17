/** What a view of the slot `plexGraph.node` draws and does: the list of the headings under its node, and what the pointer does to it. */
import { computed, watch, type Ref } from 'vue'
import { getListLayout } from '../lib/layout'
import { getMeasures } from '../lib/measures'
import { getRows } from '../lib/rows'
import type { NodeView } from '../types'
import { useAnchorBox } from './useAnchorBox'
import { useHeadings, type HeadingsSource } from './useHeadings'
import { useTypeSize } from './useTypeSize'

/** `trigger` is the element the pointer is read on, which the list hangs from. */
export function useHeadingsView(
  source: HeadingsSource,
  view: () => NodeView,
  trigger: Readonly<Ref<HTMLElement | null>>,
) {
  const { headings, enter: read, leave, choose } = useHeadings(source, () => view().path)
  const type = useTypeSize()
  const measures = computed(() => getMeasures(type.value))
  const anchor = useAnchorBox(trigger)

  const rows = computed(() => getRows(headings.value, measures.value.indentStep))
  const layout = computed(() =>
    anchor.box.value
      ? getListLayout({
          count: rows.value.length,
          limit: source.settings.headingLimit.value,
          node: anchor.box.value,
          room: anchor.room.value,
          measures: measures.value,
        })
      : null,
  )

  const enter = () => {
    anchor.measure()
    return read()
  }
  watch(
    () => [view().x, view().y, view().width, view().height, view().scale],
    anchor.measure,
    { flush: 'post' },
  )

  return { rows, layout, enter, leave, choose }
}
