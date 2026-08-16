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
  seatWord,
  type NodeStanding,
  type PlacedEdge,
  type PlacedNode,
  type PlexFrame,
  type PlexRelatedSeat,
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
    /** Draw the title a typed relationship carries. */
    showEdgeLabels?: boolean
    /** Whether reaching out is allowed at all, and so whether any node may
     *  offer a handle. */
    mayReach?: boolean
    /**
     * What to call a seat, for the one place a seat has to be written into the
     * picture: the outline a gesture draws says which one it would take.
     */
    seatName?: (seat: PlexRelatedSeat) => string
    /** A gesture in progress: where it started, where it is, what it means. */
    gestureFrom?: string | null
    gestureAt?: Point | null
    gestureOutcome?: Drop | null
  }>(),
  {
    showEdgeLabels: true,
    mayReach: true,
    seatName: seatWord,
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
  /** A handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask', id: string): void
}>()

const svg = useTemplateRef<SVGSVGElement>('svg')
defineExpose({ svg })

/**
 * One plex unit is one pixel, origin at the middle of the window.
 *
 * The viewBox is a fixed window on the drawing, centred on the focus, so a box
 * and a letter are one size however many neighbours arrive. A neighbourhood
 * too wide is clipped, and the per-seat limits and the overflow count say how
 * much.
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

/**
 * What each node is to the gesture. Only the node it left from keeps a handle
 * while one is running: the hand is somewhere else entirely, and a second
 * handle under it would offer to start a gesture already under way.
 */
const standingOf = (node: PlacedNode): NodeStanding => {
  const outcome = props.gestureOutcome
  if (outcome?.kind === 'link' && outcome.to === node.id) return 'target'
  if (props.gestureFrom === node.id) return 'source'
  return props.mayReach && props.gestureFrom === null ? 'open' : 'closed'
}

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

/**
 * The node a gesture would make, drawn where it would appear so the reader
 * sees it before letting go. A node like any other, so it is the same box in
 * the same place at the same size — it is only that it has no name yet, and
 * says the seat it would take instead, in whatever words it was given.
 */
const ghost = computed<PlacedNode | null>(() => {
  const outcome = props.gestureOutcome
  const to = props.gestureAt
  if (outcome?.kind !== 'create' || !to) return null
  return {
    id: 'ghost',
    title: props.seatName(outcome.seat),
    seat: outcome.seat,
    x: to.x,
    y: to.y,
    ...props.nodeSize,
    order: 0,
    opacity: 1,
  }
})
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
      <template v-for="edge in frame.edges" :key="`title:${edge.from}->${edge.to}`">
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
      :standing="standingOf(node)"
      @activate="emit('activate', node.id)"
      @reach="emit('reach', node.id, $event)"
      @ask="emit('ask', node.id)"
    >
      <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
    </PlexNodeView>

    <!-- The gesture itself, drawn over everything it may land on. -->
    <g v-if="thread" class="plex__reach">
      <path class="plex__thread" :d="thread" aria-hidden="true" />
      <PlexNodeView v-if="ghost" :node="ghost" standing="ghost" />
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
  user-select: none;
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
  stroke-dasharray: var(--numen-thread-dash);
}
</style>
