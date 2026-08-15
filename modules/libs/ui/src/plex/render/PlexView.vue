<script setup lang="ts">
/**
 * The drawing, and nothing else. Every number here came from `arrange/`.
 *
 * What it draws itself is the picture between the nodes: the window, the
 * edges, and the gesture crossing them. A node draws itself, and is told the
 * two things about it that only the whole picture knows.
 */
import { computed, useTemplateRef } from 'vue'
import PlexNodeView from './PlexNodeView.vue'
import {
  handleIn,
  midpointOf,
  type PlacedEdge,
  type PlacedNode,
  type PlexFrame,
  type Point,
} from '../model'
import type { Drop } from '../arrange'

const props = withDefaults(
  defineProps<{
    frame: PlexFrame
    /** The window to centre on. Measured by whoever owns the element. */
    viewport: { width: number; height: number }
    /** How big a node the gesture would make, for the shape drawn under it. */
    nodeSize: { width: number; height: number }
    /** Draw the label a typed relationship carries. */
    showEdgeLabels?: boolean
    /** The node a pointer is over, so its handle can be offered. */
    hovered?: string | null
    /** A gesture in progress: where it started, where it is, what it means. */
    gestureFrom?: string | null
    gestureAt?: Point | null
    gestureOutcome?: Drop | null
  }>(),
  {
    showEdgeLabels: true,
    hovered: null,
    gestureFrom: null,
    gestureAt: null,
    gestureOutcome: null,
  },
)

const emit = defineEmits<{
  /** A node was chosen. The identifier is the caller's, handed back as given. */
  (event: 'activate', id: string): void
  /** A gesture began at a node's handle. */
  (event: 'reach', id: string, pointer: PointerEvent): void
  (event: 'hover', id: string | null): void
}>()

const svg = useTemplateRef<SVGSVGElement>('svg')
defineExpose({ svg })

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

const offering = (node: PlacedNode) =>
  props.gestureFrom === node.id ||
  (props.gestureFrom === null && props.hovered === node.id && node.opacity >= 1)

/** The line a gesture drags behind it, from the handle to the pointer. */
const thread = computed(() => {
  const source = props.frame.nodes.find((node) => node.id === props.gestureFrom)
  const to = props.gestureAt
  if (!source || !to) return null
  const offset = handleIn(source)
  const start = { x: source.x + offset.x, y: source.y + offset.y }
  const reachOut = Math.abs(to.x - start.x) / 2
  return (
    `M ${start.x} ${start.y}` +
    ` C ${start.x + reachOut} ${start.y} ${to.x - reachOut} ${to.y} ${to.x} ${to.y}`
  )
})

/** Where a new node would appear, so the reader sees it before letting go. */
const ghost = computed(() => {
  const outcome = props.gestureOutcome
  const to = props.gestureAt
  if (outcome?.kind !== 'create' || !to) return null
  return { role: outcome.role, x: to.x, y: to.y, ...props.nodeSize }
})

/** The node a link would be made to, so it can be shown as the target. */
const aimedAt = computed(() =>
  props.gestureOutcome?.kind === 'link' ? props.gestureOutcome.to : null,
)
</script>

<template>
  <svg
    ref="svg"
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

    <PlexNodeView
      v-for="node in frame.nodes"
      :key="node.id"
      :node="node"
      :offering="offering(node)"
      :aimed="aimedAt === node.id"
      @activate="emit('activate', node.id)"
      @reach="emit('reach', node.id, $event)"
      @hover="emit('hover', $event ? node.id : null)"
    />

    <!-- The gesture itself, drawn over everything it may land on. -->
    <g v-if="thread" class="plex__reach" aria-hidden="true">
      <path class="plex__thread" :d="thread" />
      <rect
        v-if="ghost"
        class="plex__ghost"
        :style="{ '--numen-role-hue': `var(--numen-role-${ghost.role})` }"
        :x="ghost.x - ghost.width / 2"
        :y="ghost.y - ghost.height / 2"
        :width="ghost.width"
        :height="ghost.height"
      />
      <text
        v-if="ghost"
        class="plex__ghost-role"
        :x="ghost.x"
        :y="ghost.y"
        text-anchor="middle"
        dominant-baseline="central"
      >{{ ghost.role }}</text>
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

.plex__reach {
  pointer-events: none;
}

.plex__thread {
  fill: none;
  stroke: var(--numen-ring);
  stroke-width: var(--numen-edge-width);
  stroke-dasharray: 4 4;
}

.plex__ghost {
  rx: var(--numen-radius);
  fill: none;
  stroke: var(--numen-role-hue, var(--numen-ring));
  stroke-width: var(--numen-stroke);
  stroke-dasharray: 6 4;
}

.plex__ghost-role {
  fill: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
</style>
