/** What a view of the slot `plexGraph.node` draws and does: the list of the headings under its node, and what the pointer does to it. */
import { computed } from 'vue'
import { getListLayout } from '../lib/layout'
import { getMeasures } from '../lib/measures'
import { getRows } from '../lib/rows'
import type { NodeView } from '../types'
import { useHeadings, type HeadingsSource } from './useHeadings'
import { useTypeSize } from './useTypeSize'

export function useHeadingsView(source: HeadingsSource, view: () => NodeView) {
  const { headings, enter, leave, choose } = useHeadings(source, () => view().path)
  const type = useTypeSize()
  const measures = computed(() => getMeasures(type.value))

  const rows = computed(() => getRows(headings.value, measures.value.indentStep))
  const layout = computed(() =>
    getListLayout({
      count: rows.value.length,
      limit: source.settings.headingLimit.value,
      node: view(),
      measures: measures.value,
    }),
  )

  return { rows, layout, enter, leave, choose }
}
