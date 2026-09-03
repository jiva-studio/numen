<script setup lang="ts">
/**
 * The two regions the corner is read out through: one polite, and one that is
 * read over whatever else is being read. Neither is drawn.
 */
defineProps<{
  /** What is read out when the reader reaches it. */
  told: string
  /** What is read out at once. */
  cried: string
}>()
</script>

<template>
  <span class="sr-only" aria-live="polite">{{ told }}</span>
  <span class="sr-only" aria-live="assertive">{{ cried }}</span>
</template>
