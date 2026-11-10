<script setup lang="ts">
/**
 * Message shown when the window cannot reach the core.
 */
// --- Props & Emits ---
const props = defineProps<{
  failure: string
}>()

// --- State ---

// --- Handlers ---

// --- Helpers ---
</script>

<template>
  <p v-if="props.failure" class="failure">{{ props.failure }}</p>
</template>

<style scoped>
/* A failure carries filesystem paths, and a long one breaks at its current position. */
.failure {
  --max-width: 40rem;
  margin: auto;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
  max-width: var(--max-width);
  text-align: center;
  overflow-wrap: break-word;
}
</style>
