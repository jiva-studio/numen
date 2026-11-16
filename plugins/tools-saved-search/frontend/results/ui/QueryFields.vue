<script setup lang="ts">
/**
 * The field of the query of the note, filled from the note when it opens, the problems of the last
 * save, and the button that saves it.
 */
import { ref } from 'vue'
import { WORDS as words } from '../../words'
import type { SearchChange } from '../model/tab'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  id: string
  query: string
  problems: readonly string[]
  isSaving: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'save', change: SearchChange): void
}>()

/* --------------------------------- State ---------------------------------- */
const query = ref(props.query)

/* -------------------------------- Handlers -------------------------------- */
function onSave() {
  emit('save', { query: query.value })
}
</script>

<template>
  <form :id="id" class="saved-search__fields" @submit.prevent="onSave">
    <label class="saved-search__field">
      <span class="saved-search__field-label">{{ words.queryField }}</span>
      <textarea v-model="query" class="saved-search__field-input" rows="6" spellcheck="false" />
    </label>
    <p v-for="problem in problems" :key="problem" role="alert" class="saved-search__problem">
      {{ problem }}
    </p>
    <button type="submit" class="saved-search__save" :disabled="isSaving">{{ words.save }}</button>
  </form>
</template>
