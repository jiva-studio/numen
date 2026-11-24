<script setup lang="ts">
/** The numbered steps a person follows on the two devices. */

/* --------------------------------- Props ---------------------------------- */
defineProps<{ steps: readonly string[] }>()
</script>

<template>
  <ol class="link-steps">
    <li v-for="step in steps" :key="step" class="link-steps__step">{{ step }}</li>
  </ol>
</template>

<style scoped>
.link-steps {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: step;
}

.link-steps__step {
  counter-increment: step;
  display: grid;
  grid-template-columns: 1.25rem minmax(0, 1fr);
  gap: 0.5rem;
}

.link-steps__step::before {
  content: counter(step);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
  text-align: end;
}
</style>
