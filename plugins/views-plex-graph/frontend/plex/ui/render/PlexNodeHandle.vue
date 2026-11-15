<script setup lang="ts">
/**
 * The handle on a node, reached out from to make something: a disc with a cross
 * on it, drawn about its own origin and put where it belongs by the one number
 * it takes.
 *
 * pressing it must never reach that.
 */
import { HANDLE_RADIUS, type Position } from '../../lib/node'
import { isPress } from '../../lib/keys'

withDefaults(
  defineProps<{
    at: Position
    isTabStop?: boolean
  }>(),
  { isTabStop: true },
)

const emit = defineEmits<{
  /** Pressed by a pointer, which is now free to drag it somewhere. */
  (event: 'reach', pointer: PointerEvent): void
  /** Pressed from the keyboard, where there is nowhere to drag it. */
  (event: 'ask'): void
}>()

const onPointerDown = (event: PointerEvent) => {
  if (event.button !== 0 || event.ctrlKey) return
  event.stopPropagation()
  event.preventDefault()
  emit('reach', event)
}

const onKey = (event: KeyboardEvent) => {
  if (!isPress(event)) return
  event.stopPropagation()
  event.preventDefault()
  emit('ask')
}
</script>

<template>
  <g
    class="plex__handle-at"
    :style="{ '--radius': `${HANDLE_RADIUS}px` }"
    :transform="`translate(${at.x} ${at.y})`"
  >
    <circle
      class="plex__handle"
      :tabindex="isTabStop ? 0 : -1"
      role="button"
      aria-label="Reach out from here"
      @pointerdown="onPointerDown"
      @keydown="onKey"
      @click.stop
      @dblclick.stop
    />
    <rect class="plex__handle-mark plex__handle-mark--across" aria-hidden="true" />
    <rect class="plex__handle-mark plex__handle-mark--down" aria-hidden="true" />
  </g>
</template>

<style scoped>
/* The arms of the cross on the disc, and the bar they are drawn with. The
   arms read as a plus sign at about half the radius. */
.plex__handle-at {
  --arm: 0.25rem;
  --bar: 0.09375rem;
}

/* The hue is the one thing it inherits: it belongs to whatever it hangs off. */
.plex__handle {
  r: var(--radius);
  fill: var(--numen-raised);
  stroke: var(--relation-hue);
  stroke-width: var(--numen-stroke);
  cursor: crosshair;
}

.plex__handle:focus-visible {
  outline: none;
}

/* The cross, as two bars, each drawn about the origin the group is placed at. */
.plex__handle-mark {
  fill: var(--numen-ink);
  rx: calc(var(--bar) / 2);
  pointer-events: none;
}

.plex__handle-mark--across {
  x: calc(-1 * var(--arm));
  y: calc(-0.5 * var(--bar));
  width: calc(2 * var(--arm));
  height: var(--bar);
}

.plex__handle-mark--down {
  x: calc(-0.5 * var(--bar));
  y: calc(-1 * var(--arm));
  width: var(--bar);
  height: calc(2 * var(--arm));
}
</style>
