<script setup lang="ts">
/**
 * What a drag of a bar entry draws over the window: the title of the entry next to the pointer, and
 * a line where the entry goes when it is dropped.
 */
import { computed } from 'vue'
import { DragPreview, type Position } from '@numen/ui'
import type { SlotDropMarker } from '../model/slotDrag'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  label: string | null
  at: Position | null
  marker: SlotDropMarker | null
}>()

/* --------------------------------- State ---------------------------------- */
const markerStyle = computed(() =>
  props.marker
    ? {
        left: `${props.marker.x}px`,
        top: `${props.marker.top}px`,
        height: `${props.marker.height}px`,
      }
    : undefined,
)
</script>

<template>
  <DragPreview v-if="label !== null && at !== null" :at="at" :label="label" />
  <span v-if="markerStyle" class="slot-drop-marker" data-slot-drop-marker :style="markerStyle" />
</template>

<style scoped>
.slot-drop-marker {
  position: fixed;
  z-index: 50;
  inline-size: 2px;
  translate: -1px 0;
  pointer-events: none;
  background: var(--numen-accent);
}
</style>
