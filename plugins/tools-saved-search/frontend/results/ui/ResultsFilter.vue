<script setup lang="ts">
/**
 * The row above the results: the field that filters the results by their text, titles, places and
 * file names, and to its right the list of their sources.
 */
import { WORDS as words } from '../../words'
import type { SourceView } from '../lib/view'
import SourcesMenu from './SourcesMenu.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  filter: string
  sources: readonly SourceView[]
  hidden: ReadonlySet<string>
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'filter', filter: string): void
  (event: 'toggle', path: string, isShown: boolean): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event) {
  emit('filter', (event.target as HTMLInputElement).value)
}

function onToggle(path: string, isShown: boolean) {
  emit('toggle', path, isShown)
}
</script>

<template>
  <div class="saved-search__filter">
    <input
      type="search"
      class="saved-search__filter-input saved-search__field-input"
      :value="filter"
      :aria-label="words.filterResults"
      :placeholder="words.filterResults"
      @input="onInput"
    />
    <SourcesMenu :sources="sources" :hidden="hidden" @toggle="onToggle" />
  </div>
</template>
