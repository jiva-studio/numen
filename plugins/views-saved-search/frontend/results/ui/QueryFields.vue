<script setup lang="ts">
/**
 * The fields of the query, the rerank text and the limit of the note, filled from the note when they
 * open, the problems of the last save, and the button that saves them.
 */
import { ref } from 'vue'
import { WORDS as words } from '../../words'
import type { SearchChange } from '../model/tab'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  id: string
  query: string
  rerank: string
  limit: number
  problems: readonly string[]
  isSaving: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'save', change: SearchChange): void
}>()

/* --------------------------------- State ---------------------------------- */
const query = ref(props.query)
const rerank = ref(props.rerank)
const limit = ref<number | string>(props.limit > 0 ? props.limit : '')

/* -------------------------------- Handlers -------------------------------- */
function onSave() {
  emit('save', { query: query.value, rerank: rerank.value, limit: getLimit() })
}

/* --------------------------------- Helpers -------------------------------- */
function getLimit(): number {
  const value = Number(limit.value)
  return Number.isInteger(value) && value > 0 ? value : 0
}
</script>

<template>
  <form :id="id" class="saved-search__fields" @submit.prevent="onSave">
    <label class="saved-search__field">
      <span class="saved-search__field-label">{{ words.queryField }}</span>
      <textarea v-model="query" class="saved-search__field-input" rows="6" spellcheck="false" />
    </label>
    <label class="saved-search__field">
      <span class="saved-search__field-label">{{ words.rerankField }}</span>
      <input v-model="rerank" class="saved-search__field-input" type="text" />
    </label>
    <label class="saved-search__field">
      <span class="saved-search__field-label">{{ words.limitField }}</span>
      <input v-model="limit" class="saved-search__field-input" type="number" min="1" step="1" />
    </label>
    <p v-for="problem in problems" :key="problem" role="alert" class="saved-search__problem">
      {{ problem }}
    </p>
    <button type="submit" class="saved-search__save" :disabled="isSaving">{{ words.save }}</button>
  </form>
</template>
