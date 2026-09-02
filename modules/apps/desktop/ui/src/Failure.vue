<script setup lang="ts">
/**
 * What the window says when it cannot reach the core.
 *
 * Nothing else the window is showing is true while this stands, and a person
 * cannot put it away.
 */
const props = defineProps<{
  /** The core could not be reached, in words a person reads. */
  failure: string
}>()
</script>

<template>
  <p v-if="props.failure" class="failure">{{ props.failure }}</p>
</template>

<style scoped>
/* A failure carries filesystem paths, and a long one breaks where it stands. */
.failure {
  margin: auto;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
  max-width: 40rem;
  text-align: center;
  overflow-wrap: break-word;
}
</style>
