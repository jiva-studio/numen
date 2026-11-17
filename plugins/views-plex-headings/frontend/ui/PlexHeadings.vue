<script setup lang="ts">
/**
 * The view of a node of the plex that lists the headings of its note. Its box is the node but for the strips
 * the handle of the node is in, which is where the pointer is read; the list is drawn above everything
 * else, hung from the foot of the box. Clicks and drags it does not use reach the node.
 */
import type { ListLayout } from '../lib/layout'
import type { HeadingRow } from '../types'
import HeadingsList from './HeadingsList.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ rows: readonly HeadingRow[]; layout: ListLayout | null }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The pointer came onto the node or its list. */
  (event: 'enter'): void
  /** The pointer left the node and its list. */
  (event: 'leave'): void
  /** A heading was chosen, with the line of its note. */
  (event: 'choose', line: number): void
}>()
</script>

<template>
  <div class="headings" @pointerenter="emit('enter')" @pointerleave="emit('leave')">
    <Teleport to="body">
      <HeadingsList
        v-if="layout"
        :rows="rows"
        :layout="layout"
        @enter="emit('enter')"
        @leave="emit('leave')"
        @choose="emit('choose', $event)"
      />
    </Teleport>
  </div>
</template>

<style scoped>
.headings {
  --headings-handle-room: 0.75rem;
  position: absolute;
  inset-block: 0;
  inset-inline: var(--headings-handle-room);
  pointer-events: auto;
}
</style>
