<script setup lang="ts">
/**
 * The end of a sitting: nothing left to ask, and how much was answered.
 */
import { Button } from '@numen/ui'

defineProps<{ done: number }>()
defineEmits<{ (event: 'back'): void }>()
</script>

<template>
  <section class="over">
    <h1 class="over__said">Nothing left today.</h1>
    <p class="over__count">{{ done }} answered.</p>
    <Button variant="outline" @click="$emit('back')">Back to the decks</Button>
  </section>
</template>

<style scoped>
.over {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--numen-inset);
}

.over__said {
  margin: 0;
  font-size: var(--numen-display-size);
  font-weight: 600;
}

.over__count {
  margin: 0;
  color: var(--numen-edge-label);
}
</style>
