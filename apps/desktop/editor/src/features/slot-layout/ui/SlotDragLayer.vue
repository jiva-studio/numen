<script setup lang="ts">
/**
 * What a drag of a bar entry draws over the window: a copy of the entry at the pointer, and a line
 * where the entry goes when it is dropped.
 */
import { computed, useTemplateRef, watch } from 'vue'
import type { Position } from '@numen/ui'
import type { SlotDropMarker } from '../model/slotDrag'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  source: Element | null
  at: Position | null
  marker: SlotDropMarker | null
}>()

/* --------------------------------- State ---------------------------------- */
const copy = useTemplateRef<HTMLElement>('copy')

const copyStyle = computed(() => {
  if (!props.source || !props.at) return undefined
  const box = props.source.getBoundingClientRect()
  return {
    left: `${props.at.x}px`,
    top: `${props.at.y}px`,
    inlineSize: `${box.width}px`,
    blockSize: `${box.height}px`,
  }
})

const markerStyle = computed(() =>
  props.marker
    ? {
        left: `${props.marker.x}px`,
        top: `${props.marker.top}px`,
        height: `${props.marker.height}px`,
      }
    : undefined,
)

/* -------------------------------- Watchers -------------------------------- */
watch(
  [() => props.source, copy],
  ([source, element]) => {
    if (element && source) element.replaceChildren(source.cloneNode(true))
  },
  { flush: 'post' },
)
</script>

<template>
  <div v-if="copyStyle" ref="copy" class="slot-drag-copy" data-slot-drag-copy :style="copyStyle" />
  <span v-if="markerStyle" class="slot-drop-marker" data-slot-drop-marker :style="markerStyle" />
</template>

<style scoped>
.slot-drag-copy {
  position: fixed;
  z-index: var(--numen-lift-tooltip);
  translate: -50% -50%;
  pointer-events: none;
  display: flex;
  align-items: center;
  overflow: hidden;
  border-radius: var(--numen-radius-tight);
  background: var(--numen-surface);
  box-shadow: 0 2px 8px rgb(0 0 0 / 0.3);
}

.slot-drop-marker {
  position: fixed;
  z-index: var(--numen-lift-tooltip);
  inline-size: 2px;
  translate: -1px 0;
  pointer-events: none;
  background: var(--numen-accent);
}
</style>
