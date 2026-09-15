<script setup lang="ts">
/**
 * The line between two panels, and the reach around it a pointer is caught by.
 *
 * The splitter moves the panels itself while the handle is held; the handle
 * says when it is taken up and when it is put down.
 */
import { SplitterResizeHandle } from 'reka-ui'
import type { Orientation } from '../../../lib/node'

defineProps<{
  direction: Orientation
}>()

defineEmits<{
  /** The handle taken up, and put down. */
  (event: 'hold', now: boolean): void
}>()

/**
 * How far from the line a pointer is caught, in pixels, by a mouse and by a
 * finger. The splitter is told this reach and the handle draws it.
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
/* The reach hangs over both panels and stands above them, so a press on either
   side of the line is a press on the handle. */
.branch__handle {
  --line: var(--numen-stroke);
  --reach: v-bind('`${reach.fine}px`');

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

/* A line one pixel thick has no room for a ring around it, so it becomes the
   ring: the line itself is drawn in the keyboard's colour, and a pixel either
   side of it carries the same. */
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
    --reach: v-bind('`${reach.coarse}px`');
  }
}
</style>
