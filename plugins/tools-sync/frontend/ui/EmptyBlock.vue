<script setup lang="ts">
/** What a card shows in place of its rows: an icon, one title, an optional sentence and an optional action below. */
import type { Component } from 'vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ icon: Component; title: string; text?: string }>()
</script>

<template>
  <div class="empty-block">
    <component :is="icon" class="empty-block__icon" aria-hidden="true" />
    <h3 class="empty-block__title">{{ title }}</h3>
    <p v-if="text !== undefined && text !== ''" class="empty-block__text">{{ text }}</p>
    <div v-if="$slots.default" class="empty-block__action"><slot /></div>
  </div>
</template>

<style scoped>
.empty-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: calc(0.375 * var(--numen-chrome-unit));
  padding-block: calc(1.5 * var(--numen-chrome-unit));
  text-align: center;
}

.empty-block__icon {
  flex: none;
  inline-size: calc(1.75 * var(--numen-chrome-unit));
  block-size: calc(1.75 * var(--numen-chrome-unit));
  color: var(--numen-hushed);
}

.empty-block__title {
  margin: 0;
  font-size: var(--numen-chrome-text-2);
  line-height: var(--numen-line-height);
  font-weight: 600;
}

.empty-block__text {
  margin: 0;
  max-inline-size: calc(24 * var(--numen-chrome-unit));
  color: var(--numen-hushed);
  font-size: var(--numen-chrome-text-1);
  line-height: var(--numen-line-height);
  overflow-wrap: anywhere;
}

.empty-block__action {
  margin-block-start: calc(0.375 * var(--numen-chrome-unit));
}
</style>
