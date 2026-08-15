<script setup lang="ts">
/**
 * The drawing, and nothing else. Every number here came from `arrange/`.
 *
 * Labels go through a `foreignObject`: SVG text cannot wrap, cannot ellipsise
 * and does not reorder a right-to-left run.
 */
import { computed } from 'vue'
import {
  isReachable,
  midpointOf,
  nameOf,
  type PlacedEdge,
  type PlacedNode,
  type PlexFrame,
} from '../model'

const props = withDefaults(
  defineProps<{
    frame: PlexFrame
    /** The window to centre on. Measured by whoever owns the element. */
    viewport: { width: number; height: number }
    /** Draw the label a typed relationship carries. */
    showEdgeLabels?: boolean
  }>(),
  { showEdgeLabels: true },
)

const emit = defineEmits<{
  /** A node was chosen. The identifier is the caller's, handed back as given. */
  (event: 'activate', id: string): void
}>()

/**
 * One plex unit is one pixel, origin at the middle of the window.
 *
 * Fitting the viewBox to the drawing would shrink every box and letter as
 * neighbours are added, and would centre the bounding box rather than the
 * focus. A neighbourhood too wide is clipped instead; that is what the
 * per-role limits and the overflow count are for.
 */
const viewBox = computed(() => {
  const { width, height } = props.viewport
  return `${-width / 2} ${-height / 2} ${width} ${height}`
})

const path = (edge: PlacedEdge) =>
  `M ${edge.fromPoint.x} ${edge.fromPoint.y}` +
  ` C ${edge.control1.x} ${edge.control1.y}` +
  ` ${edge.control2.x} ${edge.control2.y}` +
  ` ${edge.toPoint.x} ${edge.toPoint.y}`

const activate = (node: PlacedNode) => {
  if (isReachable(node)) emit('activate', node.id)
}

const onKey = (event: KeyboardEvent, node: PlacedNode) => {
  if (event.key !== 'Enter' && event.key !== ' ') return
  event.preventDefault()
  activate(node)
}
</script>

<template>
  <svg
    class="plex"
    :viewBox="viewBox"
    preserveAspectRatio="xMidYMid meet"
    role="group"
    aria-label="Neighbourhood"
  >
    <g aria-hidden="true">
      <path
        v-for="edge in frame.edges"
        :key="`${edge.from}->${edge.to}`"
        class="plex__edge"
        :d="path(edge)"
        :opacity="edge.opacity"
      />
    </g>

    <g v-if="showEdgeLabels" aria-hidden="true">
      <template v-for="edge in frame.edges" :key="`label:${edge.from}->${edge.to}`">
        <text
          v-if="edge.label"
          class="plex__edge-label"
          :x="midpointOf(edge).x"
          :y="midpointOf(edge).y"
          :opacity="edge.opacity"
          text-anchor="middle"
          dominant-baseline="middle"
        >{{ edge.label }}</text>
      </template>
    </g>

    <g
      v-for="node in frame.nodes"
      :key="node.id"
      class="plex__node"
      :class="`plex__node--${node.role}`"
      :style="{ '--numen-role-hue': `var(--numen-role-${node.role})` }"
      :transform="`translate(${node.x} ${node.y})`"
      :opacity="node.opacity"
      :tabindex="isReachable(node) ? 0 : -1"
      :aria-hidden="isReachable(node) || node.role === 'focus' ? undefined : 'true'"
      :role="node.role === 'focus' ? 'img' : 'button'"
      :aria-label="nameOf(node)"
      @click="activate(node)"
      @keydown="onKey($event, node)"
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
          <span class="plex__label-text">{{ node.label }}</span>
        </div>
      </foreignObject>
    </g>
  </svg>
</template>

<style scoped>
.plex {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  background: var(--numen-surface);
  font-family: var(--numen-font-sans);
}

.plex__edge {
  fill: none;
  stroke: var(--numen-edge);
  stroke-width: var(--numen-edge-width);
  stroke-linecap: round;
}

.plex__edge-label {
  fill: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  paint-order: stroke;
  stroke: var(--numen-surface);
  stroke-width: var(--numen-edge-label-halo);
  stroke-linejoin: round;
}

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

.plex__label {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding-inline: 10px;
  box-sizing: border-box;
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  transition: color var(--numen-plex-move) var(--numen-easing);
}

/* Two lines, then an ellipsis: a title is a sentence often enough that one
   line throws away what distinguishes it from its neighbours. */
.plex__label-text {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
  overflow-wrap: anywhere;
  text-align: center;
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
  font-weight: 600;
}
</style>
