<script setup lang="ts">
/**
 * One fixed, ordered list of the stages of a task. Every stage keeps its row from the first frame to the last:
 * a done stage shows a check, the active one a spinner, a coming one an empty circle.
 */
import { Check, Circle } from '@lucide/vue'
import Spinner from '../spinner/Spinner.vue'
import type { Stage } from './types'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ stages: readonly Stage[] }>()
</script>

<template>
  <ol class="stage-list numen">
    <li
      v-for="(stage, at) in stages"
      :key="stage.id"
      class="stage-list__row"
      :class="`stage-list__row--${stage.state}`"
      :aria-current="stage.state === 'active' ? 'step' : undefined"
      :data-state="stage.state"
    >
      <span class="stage-list__number">{{ at + 1 }}</span>
      <span class="stage-list__marker" aria-hidden="true">
        <Check v-if="stage.state === 'done'" :size="14" />
        <Spinner v-else-if="stage.state === 'active'" />
        <Circle v-else :size="14" />
      </span>
      <span class="stage-list__label">{{ stage.label }}</span>
      <span class="stage-list__detail">{{ stage.detail ?? '' }}</span>
    </li>
  </ol>
</template>

<style scoped>
.stage-list {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.stage-list__row {
  display: grid;
  grid-template-columns: 1rem 1rem minmax(0, 1fr) auto;
  gap: 0.5rem;
  align-items: center;
  color: var(--numen-hushed);
}

.stage-list__row--done,
.stage-list__row--active {
  color: var(--numen-ink);
}

.stage-list__row--active .stage-list__label {
  font-weight: 600;
}

.stage-list__number {
  font-variant-numeric: tabular-nums;
  text-align: end;
}

.stage-list__marker {
  display: flex;
  align-items: center;
  justify-content: center;
}

.stage-list__row--active .stage-list__marker {
  color: var(--numen-accent);
}
</style>
