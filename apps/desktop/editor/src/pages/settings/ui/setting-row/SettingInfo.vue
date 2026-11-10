<script setup lang="ts">
/**
 * An info button next to a setting title; its text shows as a tooltip on hover
 * and on keyboard focus, and is the button's accessible description.
 */
import { Info } from '@lucide/vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  at: string
  label: string
  text: string
}>()
</script>

<template>
  <span class="setting-info">
    <button type="button" class="setting-info__button" :aria-label="label" :aria-describedby="at">
      <Info :size="14" aria-hidden="true" />
    </button>
    <span :id="at" role="tooltip" class="setting-info__tooltip">{{ text }}</span>
  </span>
</template>

<style scoped>
.setting-info {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
}

.setting-info__button {
  --pad: 0.125rem;
  display: inline-flex;
  padding: var(--pad);
  border: none;
  border-radius: var(--numen-radius-pill);
  background: none;
  color: var(--numen-hushed);
  cursor: help;
}

.setting-info__button:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
}

.setting-info__tooltip {
  --inset-start: 0.25rem;
  --max-width: 20rem;
  --pad: 0.375rem 0.5rem;
  position: absolute;
  inset-block-start: calc(100% + var(--inset-start));
  inset-inline-start: 0;
  z-index: 10;
  inline-size: max-content;
  max-inline-size: var(--max-width);
  padding: var(--pad);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-raised);
  color: var(--numen-ink);
  font-size: var(--numen-text-1);
  font-weight: 400;
  visibility: hidden;
}

.setting-info:hover .setting-info__tooltip,
.setting-info:focus-within .setting-info__tooltip {
  visibility: visible;
}
</style>
