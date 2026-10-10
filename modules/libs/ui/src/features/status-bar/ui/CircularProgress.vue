<script setup lang="ts">
/**
 * Circular progress indicator showing determinate percentage arc or indeterminate spinner.
 */
import { computed } from 'vue'
import type { TaskState } from '../model/types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    progress?: number | undefined
    state?: TaskState | undefined
    size?: number | undefined
    strokeWidth?: number | undefined
  }>(),
  {
    progress: undefined,
    state: 'running',
    size: 20,
    strokeWidth: 2,
  },
)

/* --------------------------------- State ---------------------------------- */
const size = computed(() => props.size ?? 20)
const strokeWidth = computed(() => props.strokeWidth ?? 2)
const state = computed(() => props.state ?? 'running')

const radius = computed(() => (size.value - strokeWidth.value) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)

const isIndeterminate = computed(() => props.progress === undefined && state.value === 'running')

const strokeDasharray = computed(() => {
  if (isIndeterminate.value) {
    return `${circumference.value * 0.3} ${circumference.value * 0.7}`
  }
  return `${circumference.value}`
})

const strokeDashoffset = computed(() => {
  if (isIndeterminate.value) return 0
  if (props.progress === undefined) return 0
  const clamped = Math.max(0, Math.min(100, props.progress))
  return circumference.value - (clamped / 100) * circumference.value
})

const containerStyle = computed(() => ({
  width: `${size.value}px`,
  height: `${size.value}px`,
}))
</script>

<template>
  <div
    class="circular-progress"
    :class="[
      `circular-progress--${state}`,
      { 'circular-progress--indeterminate': isIndeterminate },
    ]"
    :style="containerStyle"
  >
    <svg :width="size" :height="size" :viewBox="`0 0 ${size} ${size}`" aria-hidden="true">
      <circle
        class="circular-progress__track"
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        :stroke-width="strokeWidth"
      />
      <circle
        v-if="state !== 'failed'"
        class="circular-progress__indicator"
        :cx="size / 2"
        :cy="size / 2"
        :r="radius"
        :stroke-width="strokeWidth"
        :stroke-dasharray="strokeDasharray"
        :stroke-dashoffset="strokeDashoffset"
      />
      <path
        v-else
        class="circular-progress__failed-icon"
        :d="`M${size * 0.3} ${size * 0.3} L${size * 0.7} ${size * 0.7} M${size * 0.7} ${size * 0.3} L${size * 0.3} ${size * 0.7}`"
        :stroke-width="strokeWidth"
      />
    </svg>
  </div>
</template>

<style scoped>
.circular-progress {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
}

.circular-progress svg {
  transform: rotate(-90deg);
}

.circular-progress--indeterminate svg {
  animation: circular-turn 1s linear infinite;
}

.circular-progress__track {
  fill: none;
  stroke: color-mix(in srgb, var(--numen-ink) 18%, transparent);
}

.circular-progress__indicator {
  fill: none;
  stroke: var(--numen-accent);
  stroke-linecap: round;
  transition: stroke-dashoffset 0.2s ease;
}

.circular-progress--failed .circular-progress__track {
  stroke: var(--numen-alarm-bg);
}

.circular-progress__failed-icon {
  stroke: var(--numen-alarm);
  stroke-linecap: round;
  transform: rotate(90deg);
  transform-origin: center;
}

.circular-progress--cancelling .circular-progress__indicator {
  stroke: var(--numen-hushed);
}

@keyframes circular-turn {
  to {
    transform: rotate(270deg);
  }
}
</style>
