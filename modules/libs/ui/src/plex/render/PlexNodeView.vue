<script setup lang="ts">
/**
 * One node: its box, its label, and the handle to reach out from.
 *
 * Every number it draws with is already on the node it was handed. Whether a
 * pointer is over it is its own affair, so the handle appears under the hand
 * without anything above being told; what it cannot know is whether reaching
 * out is allowed here at all, and whether a gesture would land a link on it.
 *
 * The label goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import { computed, ref } from 'vue'
import { handleIn, isReachable, nameOf, type PlacedNode } from '../model'

const props = withDefaults(
  defineProps<{
    node: PlacedNode
    /** Whether reaching out from here is allowed at all. */
    mayReach?: boolean
    /** A gesture is already under way from here: the hand has moved on, and
     *  the handle it left from stays where it was. */
    reaching?: boolean
    /** A gesture in progress would land a link on this node. */
    aimed?: boolean
  }>(),
  { mayReach: true, reaching: false, aimed: false },
)

const emit = defineEmits<{
  /** Chosen, by click or by keyboard. Which node it was is the caller's to say. */
  (event: 'activate'): void
  /** A gesture began at the handle. */
  (event: 'reach', pointer: PointerEvent): void
}>()

const over = ref(false)

const activate = () => {
  if (isReachable(props.node)) emit('activate')
}

const onKey = (event: KeyboardEvent) => {
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  activate()
}

const reach = (event: PointerEvent) => {
  // The handle is inside the node, which navigates when clicked.
  event.stopPropagation()
  event.preventDefault()
  emit('reach', event)
}

/**
 * When there is a handle to press. Under the hand, or held there for as long
 * as the gesture that left from it lasts. A node on its way in or out offers
 * nothing: it is about to be somewhere else.
 */
const offering = computed(
  () =>
    props.node.opacity >= 1 && (props.reaching || (props.mayReach && over.value)),
)

/** Read three times by the drawing: the circle, and the two strokes on it. */
const handle = computed(() => handleIn(props.node))
</script>

<template>
  <g
    class="plex__node"
    :style="{ '--numen-role-hue': `var(--numen-role-${node.role})` }"
    :transform="`translate(${node.x} ${node.y})`"
    :opacity="node.opacity"
    :tabindex="isReachable(node) ? 0 : -1"
    :aria-hidden="isReachable(node) || node.role === 'focus' ? undefined : 'true'"
    :role="node.role === 'focus' ? 'img' : 'button'"
    :class="[`plex__node--${node.role}`, { 'plex__node--aimed': aimed }]"
    :aria-label="nameOf(node)"
    @click="activate"
    @keydown="onKey"
    @pointerenter="over = true"
    @pointerleave="over = false"
  >
    <rect
      class="plex__box"
      :x="-node.width / 2"
      :y="-node.height / 2"
      :width="node.width"
      :height="node.height"
    />
    <foreignObject
      :x="-node.width / 2"
      :y="-node.height / 2"
      :width="node.width"
      :height="node.height"
    >
      <div class="plex__label">
        <!-- Whatever stands for the thing a node addresses. The plex has no
             way to know what that is, so it is handed one. -->
        <span v-if="$slots.icon" class="plex__icon" aria-hidden="true">
          <slot name="icon" />
        </span>
        <span class="plex__label-text">{{ node.label }}</span>
      </div>
    </foreignObject>

    <!-- Reach out from here to make something. Under the hand so it is there
         when wanted and out of the way when not. -->
    <template v-if="offering">
      <circle
        class="plex__handle"
        :cx="handle.x"
        :cy="handle.y"
        r="9"
        role="button"
        aria-label="Reach out from here"
        @pointerdown="reach"
        @click.stop
      />
      <path
        class="plex__handle-mark"
        :d="`M ${handle.x - 4} ${handle.y} h 8 M ${handle.x} ${handle.y - 4} v 8`"
        aria-hidden="true"
      />
    </template>
  </g>
</template>

<style scoped>
/* No transition on the position — it comes from the frame, and a CSS one here
   would race it. Hover is a filter because fill is spoken for by the role. */
.plex__node {
  cursor: pointer;
  transition: filter var(--numen-motion-hover) var(--numen-easing);
}

.plex__node:hover {
  filter: brightness(var(--numen-hover-brightness));
}

.plex__node--focus {
  cursor: default;
}

/* The hue comes from the node's own role, so a new role needs a token and
   nothing here. It changes over the length of the move that changes the role,
   or a node would wear the focus colours while still halfway there. */
.plex__box {
  rx: var(--numen-radius);
  fill: var(--numen-node-bg);
  stroke: var(--numen-role-hue, var(--numen-node-border));
  stroke-width: var(--numen-stroke);
  transition:
    fill var(--numen-plex-move) var(--numen-easing),
    stroke var(--numen-plex-move) var(--numen-easing);
}

/* Icon then label, centred together in a box of a size the arrangement chose.

   Not selectable: a label is something to look at and press, and a drag that
   paints it blue is a drag that was meant to reach somewhere. */
.plex__label {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding-inline: 10px;
  box-sizing: border-box;
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  user-select: none;
  transition: color var(--numen-plex-move) var(--numen-easing);
}

.plex__icon {
  flex: none;
  display: flex;
  align-items: center;
  color: var(--numen-role-hue, var(--numen-node-fg));
}

/* One line, then an ellipsis. Two lines cost as much height again for a title
   that is a sentence, and a box that grows is a box the arrangement did not
   plan for. */
.plex__label-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plex__handle {
  fill: var(--numen-node-bg);
  stroke: var(--numen-role-hue, var(--numen-node-border));
  stroke-width: var(--numen-stroke);
  cursor: crosshair;
}

.plex__handle-mark {
  stroke: var(--numen-node-fg);
  stroke-width: 1.5;
  stroke-linecap: round;
  pointer-events: none;
}

/* The node a link would be made to, while the pointer is still on it. */
.plex__node--aimed .plex__box {
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

.plex__node:focus-visible {
  outline: none;
}

.plex__node:focus-visible .plex__box {
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

.plex__node--focus .plex__box {
  rx: var(--numen-radius-focus);
  fill: var(--numen-focus-bg);
  stroke: var(--numen-focus-border);
}

.plex__node--focus .plex__label {
  color: var(--numen-focus-fg);
}
</style>
