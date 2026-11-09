<script setup lang="ts">
/**
 * What a drag of a bar entry draws over the window: a copy of the entry that follows the pointer.
 * The entry itself stands, dimmed, where a drop would put it.
 */
import { computed, useTemplateRef, watch } from 'vue'
import type { Position } from '@numen/ui'
import type { SlotDragCopy } from '../model/slotDrag'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  copy: SlotDragCopy | null
  at: Position | null
}>()

/* --------------------------------- State ---------------------------------- */
const host = useTemplateRef<HTMLElement>('host')

const hostStyle = computed(() =>
  props.copy && props.at
    ? {
        left: `${props.at.x}px`,
        top: `${props.at.y}px`,
        inlineSize: `${props.copy.width}px`,
        blockSize: `${props.copy.height}px`,
      }
    : undefined,
)

/* -------------------------------- Watchers -------------------------------- */
watch(
  [() => props.copy, host],
  ([copy, element]) => {
    if (element && copy) element.replaceChildren(copy.node)
  },
  { flush: 'post' },
)
</script>

<template>
  <div v-if="hostStyle" ref="host" class="slot-drag-copy" data-slot-drag-copy :style="hostStyle" />
</template>

<style>
[data-numen-slot-placeholder] {
  opacity: 0.35;
}
</style>

<style scoped>
.slot-drag-copy {
  position: fixed;
  z-index: var(--numen-lift-tooltip);
  display: flex;
  align-items: center;
  translate: -50% -50%;
  pointer-events: none;
}
</style>
