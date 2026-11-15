<script setup lang="ts">
/**
 * One result: a button holding the title of its file, its path when the title is not the file name,
 * its place and its text. Activating it opens the file at the place.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  title: string
  path: string
  place: string
  text: string
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
  <button type="button" class="saved-search__result" @click="onOpen">
    <span class="saved-search__result-title">{{ title }}</span>
    <span v-if="path" class="saved-search__result-path">{{ path }}</span>
    <span v-if="place" class="saved-search__result-place">{{ place }}</span>
    <span class="saved-search__result-text">{{ text }}</span>
  </button>
</template>
