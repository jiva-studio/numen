<script setup lang="ts">
/**
 * The line between two panels, and the hit area around it that captures the
 * pointer.
 *
 * The splitter moves the panels itself while the handle is held; the handle
 * reports when dragging starts and ends.
 */
import { SplitterResizeHandle } from 'reka-ui'
import type { Orientation } from '../../../lib/node'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  direction: Orientation
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  /** The handle taken up, and put down. */
  (event: 'hold', now: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
/**
 * The distance in pixels from the line within which a pointer is captured, for
 * a mouse and for touch. The splitter receives this distance and the handle
 * renders it.
 */
const reach = { fine: 7, coarse: 15 }
</script>

<template>
  <SplitterResizeHandle
    class="branch__handle"
    aria-label="Resize panes"
    :data-direction="direction"
    :hit-area-margins="reach"
    @dragging="$emit('hold', $event)"
  />
</template>

<style scoped>
/* The hit area extends over both panels and is stacked above them. */
/* A press on either side of the line is a press on the handle. */
.branch__handle {
  --line: var(--numen-stroke);
  --reach: 7px;

  position: relative;
  z-index: 1;
  flex: none;
  background: var(--numen-rule);
}

/* The splitter draws the pointer as a double arrow everywhere its reach is
   caught, and the line under it carries that same arrow. */
.branch__handle[data-direction='horizontal'] {
  inline-size: var(--line);
  cursor: ew-resize;
}

.branch__handle[data-direction='vertical'] {
  block-size: var(--line);
  cursor: ns-resize;
}

/* A one-pixel line has no space for a focus ring around it, so the line itself becomes the ring. */
/* It is drawn in the focus color, and one pixel on each side carries the same color. */
.branch__handle:focus-visible {
  outline: none;
  background: var(--numen-ring);
  box-shadow: 0 0 0 var(--numen-stroke) var(--numen-ring);
}

.branch__handle::after {
  content: '';
  position: absolute;
  inset-block: 0;
  inset-inline: calc(var(--reach) * -1);
}

.branch__handle[data-direction='vertical']::after {
  inset-inline: 0;
  inset-block: calc(var(--reach) * -1);
}

.branch__handle[data-state='drag'] {
  background: var(--numen-ring);
}

/* A finger is caught from further out than a pointer. */
@media (pointer: coarse) {
  .branch__handle {
    --reach: 15px;
  }
}
</style>
