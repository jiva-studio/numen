<script setup lang="ts">
/**
 * The view of a node of the plex that lists the headings of its note. Its box is the whole of the node, which
 * is where the pointer is read; the list hangs from the foot of the box. Clicks and drags it does not use reach
 * the node.
 */
import type { ListLayout } from '../lib/layout'
import type { HeadingRow } from '../types'
import HeadingsList from './HeadingsList.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ rows: readonly HeadingRow[]; layout: ListLayout | null }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The pointer came onto the node. */
  (event: 'enter'): void
  /** The pointer left the node and its list. */
  (event: 'leave'): void
  /** A heading was chosen, with the line of its note. */
  (event: 'choose', line: number): void
}>()
</script>

<template>
  <div class="headings" @pointerenter="emit('enter')" @pointerleave="emit('leave')">
    <HeadingsList v-if="layout" :rows="rows" :layout="layout" @choose="emit('choose', $event)" />
  </div>
</template>

<style scoped>
.headings {
  position: absolute;
  inset: 0;
  pointer-events: auto;
}
</style>
