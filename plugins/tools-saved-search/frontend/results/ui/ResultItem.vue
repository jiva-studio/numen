<script setup lang="ts">
/**
 * One result: a button holding the title of its file, its path when the title is not the file name,
 * its place and its text; for a whole file, its title, path and kind. Activating it opens the file at
 * the place, or a whole file at its start.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  title: string
  path: string
  place: string
  text: string
  kind: string
  isWholeFile: boolean
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open'): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onOpen() {
  emit('open')
}
</script>

<template>
  <button
    type="button"
    class="saved-search__result"
    :class="{ 'saved-search__result--file': isWholeFile }"
    @click="onOpen"
  >
    <span class="saved-search__result-head">
      <span class="saved-search__result-title">{{ title }}</span>
      <span v-if="path" class="saved-search__result-path">{{ path }}</span>
      <span v-if="isWholeFile" class="saved-search__result-kind">{{ kind }}</span>
    </span>
    <span v-if="place" class="saved-search__result-place">{{ place }}</span>
    <span v-if="text" class="saved-search__result-text">{{ text }}</span>
  </button>
</template>
