<script setup lang="ts">
/**
 * The button that opens the list of the files the results come from, each with a checkbox that shows
 * or hides the results of that file.
 */
import { ref, useId } from 'vue'
import { WORDS as words } from '../../words'
import type { SourceView } from '../lib/view'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  sources: readonly SourceView[]
  hidden: ReadonlySet<string>
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'toggle', path: string, isShown: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const listId = useId()

/* -------------------------------- Handlers -------------------------------- */
function onToggleOpen() {
  isOpen.value = !isOpen.value
}

function onChange(path: string, event: Event) {
  emit('toggle', path, (event.target as HTMLInputElement).checked)
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') isOpen.value = false
}
</script>

<template>
  <div class="saved-search__sources" @keydown="onKeydown">
    <button
      type="button"
      class="saved-search__sources-toggle"
      :aria-expanded="isOpen"
      :aria-controls="listId"
      @click="onToggleOpen"
    >
      {{ words.sources }}
    </button>
    <ul
      v-if="isOpen"
      :id="listId"
      class="saved-search__sources-list"
      :aria-label="words.sources"
    >
      <li v-for="source in sources" :key="source.path" class="saved-search__sources-item">
        <label class="saved-search__source">
          <input
            type="checkbox"
            :checked="!hidden.has(source.path)"
            @change="onChange(source.path, $event)"
          />
          <span class="saved-search__source-title">{{ source.title }}</span>
          <span class="saved-search__source-path">{{ source.path }}</span>
        </label>
      </li>
    </ul>
  </div>
</template>
