/** The layout of the row of pages in the viewport, and what of it is drawn. */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { Size } from '@numen/ui'
import { GAP, SHARP, getCurrentPage, getPagesWithin, row, type Page, type Row } from '../lib/strip'

export interface RowLayoutSources {
  readonly getPages: () => readonly Page[]
  readonly viewport: Ref<Size>
  readonly zoom: Ref<number>
  readonly offset: Ref<number>
}

export interface RowLayout {
  readonly pageLayout: ComputedRef<Row>
  readonly visiblePages: ComputedRef<number[]>
  readonly sharpPages: ComputedRef<ReadonlySet<number>>
  readonly middle: ComputedRef<number>
  readonly rowStyle: ComputedRef<Record<string, string>>
  readonly getBoxStyle: (page: number) => Record<string, string>
}

export function useRowLayout({ getPages, viewport, zoom, offset }: RowLayoutSources): RowLayout {
  const pageLayout = computed(() => row(getPages(), viewport.value, zoom.value))
  const visiblePages = computed(() =>
    getPagesWithin(pageLayout.value, viewport.value, offset.value),
  )
  const sharpPages = computed(
    () => new Set(getPagesWithin(pageLayout.value, viewport.value, offset.value, SHARP)),
  )
  const middle = computed(() => getCurrentPage(pageLayout.value, viewport.value, offset.value))
  const rowStyle = computed(() => ({
    inlineSize: `${pageLayout.value.length}px`,
    blockSize: `${pageLayout.value.height + 2 * GAP}px`,
  }))

  const getBoxStyle = (page: number) => ({
    insetInlineStart: `${pageLayout.value.starts[page] ?? 0}px`,
    insetBlockStart: `${GAP}px`,
    inlineSize: `${pageLayout.value.widths[page] ?? 0}px`,
    blockSize: `${pageLayout.value.height}px`,
  })

  return { pageLayout, visiblePages, sharpPages, middle, rowStyle, getBoxStyle }
}
