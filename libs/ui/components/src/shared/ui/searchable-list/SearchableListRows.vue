<script setup lang="ts">
/**
 * The rows of a searchable list, scrolling inside their frame, each group
 * under its name. Rows that are checked are one line each, under no group name, and only
 * those in sight are drawn.
 */
import { computed, useTemplateRef } from 'vue'
import { measureElementRect } from '@/shared/api/dom'
import SearchableListOption from './SearchableListOption.vue'
import { getHeadings, type ListRow } from './rows'
import { useRowWindow } from './useRowWindow'

const props = defineProps<{
  rows: readonly ListRow[]
  activeIndex: number
  current: readonly string[]
  listId: string
  label: string
  isMultiple?: boolean
}>()

defineEmits<{
  /** A row was pressed. */
  (event: 'choose', key: string): void
  /** The pointer moved onto a row. */
  (event: 'hover', key: string): void
}>()

defineExpose({ scrollToActive })

const list = useTemplateRef<HTMLElement>('list')

const { first, shown, windowStyle, onScroll, scrollToRow } = useRowWindow({
  list,
  rows: () => props.rows,
  isMultiple: () => props.isMultiple === true,
})

/** The group name drawn over each row, where there is one. */
const headings = computed(() => getHeadings(props.rows))

function scrollToActive(): void {
  if (props.isMultiple) {
    scrollToRow(props.activeIndex)
    return
  }
  const view = list.value
  const row = view?.querySelector('[aria-selected="true"]')
  if (!view || !row || view.scrollHeight <= view.clientHeight) return
  const rowBox = measureElementRect(row)
  const viewBox = measureElementRect(view)
  if (rowBox.top < viewBox.top) view.scrollTop -= viewBox.top - rowBox.top
  else if (rowBox.bottom > viewBox.bottom) view.scrollTop += rowBox.bottom - viewBox.bottom
}
</script>

<template>
  <div
    :id="listId"
    ref="list"
    class="searchable-list__list"
    role="listbox"
    :aria-label="label"
    :aria-multiselectable="isMultiple || undefined"
    :data-multiple="isMultiple"
    @scroll.passive="onScroll"
  >
    <div :style="windowStyle">
      <SearchableListOption
        v-for="(row, offset) in shown"
        :id="`${listId}-${first + offset}`"
        :key="row.key"
        :row="row"
        :heading="isMultiple ? undefined : headings[first + offset]"
        :has-separator="
          !isMultiple && offset > 0 && row.kind === 'item' && row.item.hasSeparator === true
        "
        :is-active="first + offset === activeIndex"
        :is-current="!isMultiple && row.kind === 'item' && current.includes(row.item.id)"
        :is-multiple="isMultiple"
        @choose="$emit('choose', row.key)"
        @hover="$emit('hover', row.key)"
      />
    </div>
  </div>
</template>

<style scoped>
.searchable-list__list {
  box-sizing: border-box;
  max-block-size: calc(var(--rows) * var(--row) + 2 * var(--inset));
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: var(--inset);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

/* The rows are placed by their height, so nothing is added above or below them. */
.searchable-list__list[data-multiple='true'] {
  max-block-size: calc(var(--rows) * var(--row));
  padding-block: 0;
}
</style>
