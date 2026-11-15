<script setup lang="ts">
/**
 * The query of the note as written, read only, and the rerank text after its label when there is one.
 */

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  query: string
  rerank: string
  rerankLabel: string
}>()
</script>

<template>
  <div class="saved-search__query">
    <pre class="saved-search__query-text">{{ query }}</pre>
    <p v-if="rerank" class="saved-search__rerank">
      <span class="saved-search__rerank-label">{{ rerankLabel }}</span>
      {{ rerank }}
    </p>
  </div>
</template>
