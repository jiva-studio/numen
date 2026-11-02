<script setup lang="ts">
/**
 * The rows of a searchable list, scrolling inside their frame, each group
 * under its name.
 */
import { computed, useTemplateRef } from 'vue'
import { measureElementRect } from '@/shared/lib/dom'
import SearchableListOption from './SearchableListOption.vue'
import { getHeadings, type ListRow } from './rows'

const props = defineProps<{
  rows: readonly ListRow[]
  /** The row the keyboard is on. */
  activeIndex: number
  /** The identifier of the item in force. */
  current: string | null
  /** The identifier of the list, which the ones of its rows are made from. */
  listId: string
  label: string
}>()

defineEmits<{
  /** A row was pressed. */
  (event: 'choose', key: string): void
  /** The pointer moved onto a row. */
  (event: 'hover', key: string): void
}>()

const list = useTemplateRef<HTMLElement>('list')

/** The group name drawn over each row, where there is one. */
const headings = computed(() => getHeadings(props.rows))

/**
 * Brings the row the keyboard is on into view by scrolling the list alone: scrolling the page
 * to a row the frame still hides would move whatever the frame is placed over.
 */
const scrollToActive = (): void => {
  const view = list.value
  const row = view?.querySelector('[aria-selected="true"]')
  if (!view || !row || view.scrollHeight <= view.clientHeight) return
  const rowBox = measureElementRect(row)
  const viewBox = measureElementRect(view)
  if (rowBox.top < viewBox.top) view.scrollTop -= viewBox.top - rowBox.top
  else if (rowBox.bottom > viewBox.bottom) view.scrollTop += rowBox.bottom - viewBox.bottom
}

defineExpose({ scrollToActive })
</script>

<template>
  <div :id="listId" ref="list" class="searchable-list__list" role="listbox" :aria-label="label">
    <SearchableListOption
      v-for="(row, index) in rows"
      :id="`${listId}-${index}`"
      :key="row.key"
      :row="row"
      :heading="headings[index]"
      :is-active="index === activeIndex"
      :is-current="row.kind === 'item' && row.item.id === current"
      @choose="$emit('choose', row.key)"
      @hover="$emit('hover', row.key)"
    />
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
</style>
