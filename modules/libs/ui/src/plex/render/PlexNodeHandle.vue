<script setup lang="ts">
/**
 * The handle on a node, reached out from to make something: a disc with a
 * cross on it.
 *
 * Drawn about its own origin and put where it belongs by the one number it
 * takes, so every size in it is a token and none is arithmetic. It sits inside
 * a node that navigates when clicked, so pressing it must never reach that.
 */
import type { Point } from '../model'
import { isPress } from './keys'

defineProps<{
  /** Where it sits, in the coordinates of whatever draws it. */
  at: Point
}>()

const emit = defineEmits<{
  /** Pressed by a pointer, which is now free to drag it somewhere. */
  (event: 'reach', pointer: PointerEvent): void
  /** Pressed from the keyboard, where there is nowhere to drag it. */
  (event: 'ask'): void
}>()

const onPointerDown = (event: PointerEvent) => {
  event.stopPropagation()
  event.preventDefault()
  emit('reach', event)
}

const onKey = (event: KeyboardEvent) => {
  if (!isPress(event)) return
  event.preventDefault()
  emit('ask')
}
</script>

<template>
  <g class="plex__handle-at" :transform="`translate(${at.x} ${at.y})`">
    <circle
      class="plex__handle"
      tabindex="0"
      role="button"
      aria-label="Reach out from here"
      @pointerdown="onPointerDown"
      @keydown.stop="onKey"
      @click.stop
    />
    <rect class="plex__handle-mark plex__handle-mark--across" aria-hidden="true" />
    <rect class="plex__handle-mark plex__handle-mark--down" aria-hidden="true" />
  </g>
</template>

<style scoped>
/* The hue is the one thing it inherits: it belongs to whatever it hangs off. */
.plex__handle {
  r: var(--numen-handle-radius);
  fill: var(--numen-node-bg);
  stroke: var(--numen-role-hue, var(--numen-node-border));
  stroke-width: var(--numen-stroke);
  cursor: crosshair;
}

.plex__handle:focus-visible {
  outline: none;
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

/* The cross, as two bars rather than a stroked path, so its length and its
   thickness are sizes the theme sets and not numbers in a `d`. */
.plex__handle-mark {
  fill: var(--numen-node-fg);
  rx: calc(var(--numen-handle-stroke) / 2);
  pointer-events: none;
}

.plex__handle-mark--across {
  x: calc(-1 * var(--numen-handle-arm));
  y: calc(-0.5 * var(--numen-handle-stroke));
  width: calc(2 * var(--numen-handle-arm));
  height: var(--numen-handle-stroke);
}

.plex__handle-mark--down {
  x: calc(-0.5 * var(--numen-handle-stroke));
  y: calc(-1 * var(--numen-handle-arm));
  width: var(--numen-handle-stroke);
  height: calc(2 * var(--numen-handle-arm));
}
</style>
