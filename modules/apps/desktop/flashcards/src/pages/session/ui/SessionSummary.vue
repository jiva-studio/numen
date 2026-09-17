<script setup lang="ts">
/**
 * The end of a session: nothing left to ask today, and what was answered.
 */
import { Button } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  /** How many answers this session wrote. */
  done: number
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{ (event: 'leave'): void }>()
</script>

<template>
  <section class="finished">
    <h1 class="finished__said">Nothing left today.</h1>
    <p class="finished__count">{{ done }} answered.</p>
    <Button variant="outline" @click="$emit('leave')">Back to the decks</Button>
  </section>
</template>

<style scoped>
.finished {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--numen-inset);
}

.finished__said {
  margin: 0;
  font-size: var(--numen-display-size);
  font-weight: 600;
}

.finished__count {
  margin: 0;
  color: var(--numen-hushed);
}
</style>
