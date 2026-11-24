<script setup lang="ts">
/** What a card shows when it has no rows: an icon, one title and one sentence of what to do. */
import type { Component } from 'vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ icon: Component; title: string; text: string }>()
</script>

<template>
  <div class="empty-block">
    <component :is="icon" class="empty-block__icon" :size="28" aria-hidden="true" />
    <h3 class="empty-block__title">{{ title }}</h3>
    <p class="empty-block__text">{{ text }}</p>
  </div>
</template>

<style scoped>
.empty-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.375rem;
  padding-block: 1.5rem;
  text-align: center;
}

.empty-block__icon {
  color: var(--numen-hushed);
}

.empty-block__title {
  margin: 0;
  font-size: var(--numen-chrome-text-2);
  font-weight: 600;
}

.empty-block__text {
  margin: 0;
  max-inline-size: 24rem;
  color: var(--numen-hushed);
}
</style>
