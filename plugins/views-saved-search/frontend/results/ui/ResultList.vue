<script setup lang="ts">
/**
 * The results of a saved search, best first, each one button that opens its file at its place.
 */
import type { ResultItemView } from '../lib/view'
import ResultItem from './ResultItem.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  items: readonly ResultItemView[]
  label: string
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open', index: number): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onOpen(index: number) {
  emit('open', index)
}
</script>

<template>
  <ol class="saved-search__results" :aria-label="label">
    <li v-for="(item, index) in items" :key="index" class="saved-search__results-item">
      <ResultItem
        :title="item.title"
        :path="item.path"
        :place="item.place"
        :text="item.text"
        @open="onOpen(index)"
      />
    </li>
  </ol>
</template>
