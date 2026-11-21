<script setup lang="ts">
/** One heading of the list, in a row of its own. What choosing it does is the caller's. */
import { computed } from 'vue'
import type { HeadingRow } from '../types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ row: HeadingRow; height: number }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The heading was chosen. */
  (event: 'choose'): void
}>()

/* --------------------------------- State ---------------------------------- */
const style = computed(() => ({
  '--row-height': `${props.height}px`,
  '--row-indent': `${props.row.indent}px`,
}))
</script>

<template>
  <div class="headings__row" :style="style" @click.stop="emit('choose')" @dblclick.stop>
    <span class="headings__text">{{ row.text }}</span>
  </div>
</template>

<style scoped>
.headings__row {
  --row-radius: 0.1875rem;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  block-size: var(--row-height);
  padding-inline-start: calc(var(--headings-pad-inline) + var(--row-indent));
  padding-inline-end: var(--headings-pad-inline);
  border-radius: var(--row-radius);
  color: var(--numen-ink);
  cursor: pointer;
  transition: background var(--numen-motion-hover) var(--numen-easing);
}

.headings__row:hover {
  background: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 12%);
}

.headings__text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
