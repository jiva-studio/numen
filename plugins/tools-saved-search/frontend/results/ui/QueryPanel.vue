<script setup lang="ts">
/**
 * The query of the note as written, read only.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  query: string
}>()
</script>

<template>
  <div class="saved-search__query">
    <pre class="saved-search__query-text">{{ query }}</pre>
  </div>
</template>
