<script setup lang="ts">
/**
 * A ring that shows how far something has come: filled to the progress where it is known, a short
 * turning arc where it is not. The large ring writes the percent inside.
 */
import { computed } from 'vue'
import { getRingView, RING_LENGTH } from './ringView'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    /** From 0 to 1; undefined where the progress is not known. */
    value?: number | undefined
    size?: 'small' | 'large'
    /** The name of what progresses; without it the ring is decorative. */
    label?: string
  }>(),
  { size: 'small' },
)

/* --------------------------------- State ---------------------------------- */
const view = computed(() => getRingView(props.value))
const hasText = computed(() => props.size === 'large' && view.value.isKnown)
</script>

<template>
  <span
    class="progress-ring numen"
    :data-size="size"
    :data-known="view.isKnown"
    :role="label ? 'progressbar' : undefined"
    :aria-hidden="label ? undefined : 'true'"
    :aria-label="label"
    :aria-valuemin="label ? 0 : undefined"
    :aria-valuemax="label ? 100 : undefined"
    :aria-valuenow="label && view.isKnown ? view.percent : undefined"
  >
    <svg class="progress-ring__svg" viewBox="0 0 32 32" aria-hidden="true">
      <circle class="progress-ring__track" cx="16" cy="16" r="14" fill="none" />
      <circle
        class="progress-ring__arc"
        cx="16"
        cy="16"
        r="14"
        fill="none"
        :pathLength="RING_LENGTH"
        :stroke-dasharray="view.dash"
      />
    </svg>
    <span v-if="hasText" class="progress-ring__text">{{ view.percent }}%</span>
  </span>
</template>

<style scoped>
.progress-ring {
  --ring-small: 1rem;
  --ring-large: 4rem;
  --ring-size: var(--ring-small);
  --ring-thickness: 3px;

  position: relative;
  display: inline-grid;
  flex: none;
  place-items: center;
  inline-size: var(--ring-size);
  block-size: var(--ring-size);
}

.progress-ring[data-size='large'] {
  --ring-size: var(--ring-large);
}

.progress-ring__svg {
  position: absolute;
  inset: 0;
  inline-size: 100%;
  block-size: 100%;
  transform: rotate(-90deg);
}

.progress-ring[data-known='false'] .progress-ring__svg {
  animation: progress-ring-turn var(--numen-wait-spin) linear infinite;
}

.progress-ring__track,
.progress-ring__arc {
  stroke-width: var(--ring-thickness);
}

.progress-ring__track {
  stroke: currentColor;
  opacity: 0.2;
}

.progress-ring__arc {
  stroke: currentColor;
  stroke-linecap: round;
}

.progress-ring__text {
  position: relative;
  font-size: var(--numen-text-2);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--numen-ink);
}

@keyframes progress-ring-turn {
  from {
    transform: rotate(-90deg);
  }

  to {
    transform: rotate(270deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .progress-ring[data-known='false'] .progress-ring__svg {
    animation: none;
  }
}
</style>
