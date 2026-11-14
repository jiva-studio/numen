<script setup lang="ts">
/**
 * The accordion at the top of the tab: the button that opens and closes the fields of the query.
 */
import { useId } from 'vue'
import { WORDS as words } from '../../words'
import type { SearchChange } from '../model/tab'
import QueryFields from './QueryFields.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  isOpen: boolean
  query: string
  rerank: string
  limit: number
  problems: readonly string[]
  isSaving: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'toggle'): void
  (event: 'save', change: SearchChange): void
}>()

/* --------------------------------- State ---------------------------------- */
const fieldsId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  emit('toggle')
}

function onSave(change: SearchChange) {
  emit('save', change)
}
</script>

<template>
  <div class="saved-search__editor">
    <button
      type="button"
      class="saved-search__editor-toggle"
      :aria-expanded="isOpen"
      :aria-controls="fieldsId"
      @click="onToggle"
    >
      {{ words.editQuery }}
    </button>
    <QueryFields
      v-if="isOpen"
      :id="fieldsId"
      :query="query"
      :rerank="rerank"
      :limit="limit"
      :problems="problems"
      :is-saving="isSaving"
      @save="onSave"
    />
  </div>
</template>
