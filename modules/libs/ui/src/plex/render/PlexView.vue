<script setup lang="ts">
/**
 * The drawing, and nothing else. Every number here came from `arrange/`.
 *
 * What it draws itself is the picture between the nodes: the window, the
 * edges, and the gesture crossing them. A node draws itself, and is told the
 * two things about it that only the whole picture knows.
 */
import { computed, ref, useId, useTemplateRef, watch } from 'vue'
import PlexNodeView from './PlexNodeView.vue'
import {
  edgeKey,
  handleIn,
  midpointOf,
  seatWord,
  type NodeStanding,
  type PlacedEdge,
  type PlacedNode,
  type PlexFrame,
  type PlexRelatedSeat,
  type PlexShowing,
  type Point,
} from '../model'
import type { Drop } from '../arrange'
import type { MenuOpening } from '../../menu/model'

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
  /** A node was asked for on its own, and where it is to be drawn. */
  (event: 'show', id: string, showing: PlexShowing): void
  /** A gesture began at a node's handle. */
  (event: 'reach', id: string, pointer: PointerEvent): void
  /** A handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask', id: string): void
  /** A menu was asked for on a node: where, from what, and by what. */
  (event: 'menu', id: string, at: Point, from: SVGGElement, opening: MenuOpening): void
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

/** The same curve, running the way its words are read. */
const readingLine = (edge: PlacedEdge) =>
  edge.heading === 'left'
    ? `M ${edge.toPoint.x} ${edge.toPoint.y}` +
      ` C ${edge.control2.x} ${edge.control2.y}` +
      ` ${edge.control1.x} ${edge.control1.y}` +
      ` ${edge.fromPoint.x} ${edge.fromPoint.y}`
    : path(edge)

/** Two plexes on one page each name their own paths. */
const uid = useId()

/**
 * Every edge with what the drawing asks of it: the two keys it is remembered
 * by, the curve, and the line its title is set along. A title follows the curve
 * where it holds one direction across the page, and sits flat at the midpoint
 * where it does not.
 */
const lines = computed(() =>
  props.frame.edges.map((edge, at) => ({
    edge,
    /** One drawing per pair and direction, so a pair may carry two lines. */
    key: `${edge.from}->${edge.to}`,
    /** What the hand is on: two lines between one pair are one line to point at. */
    pair: edgeKey(edge),
    d: path(edge),
    titlePath: edge.label && edge.heading !== 'none' ? `${uid}-title-${at}` : null,
    titleLine: readingLine(edge),
  })),
)

/** The edge the hand is on, by a key that survives the re-routing of a move. */
const over = ref<string | null>(null)

// A new frame is a picture on its way somewhere, and the hand is on none of
// it until it settles.
watch(
  () => props.frame,
  () => {
    over.value = null
  },
)

/** The lines drawn over the boxes, and every other, drawn under them. */
const lifted = computed(() => lines.value.filter((line) => line.pair === over.value))

const resting = computed(() => lines.value.filter((line) => line.pair !== over.value))

/**
 * A title is painted twice over, the halo finished before a letter is drawn.
 * Each glyph set along a path is a run of its own, and a halo painted with the
 * letters lies over the one beside it.
 */
const TITLE_LAYERS = ['halo', 'letters'] as const

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
    <defs v-if="showEdgeLabels">
      <template v-for="line in lines" :key="`along:${line.key}`">
        <path v-if="line.titlePath" :id="line.titlePath" :d="line.titleLine" />
      </template>
    </defs>

    <g aria-hidden="true">
      <path
        v-for="line in resting"
        :key="line.key"
        class="plex__edge"
        :d="line.d"
        :opacity="line.edge.opacity"
      />
    </g>

    <!-- The band a line is found by. It paints nothing, and the nodes come
         after it, so a box under the hand is what the hand is on. -->
    <g aria-hidden="true">
      <path
        v-for="line in lines"
        :key="`reach:${line.key}`"
        class="plex__edge-hit"
        :d="line.d"
        @pointerenter="over = line.pair"
        @pointerleave="over = null"
      />
    </g>

    <g v-if="showEdgeLabels" aria-hidden="true">
      <template v-for="line in resting" :key="`title:${line.key}`">
        <template v-for="layer in TITLE_LAYERS" :key="layer">
          <text
            v-if="line.titlePath"
            class="plex__edge-label"
            :class="`plex__edge-label--${layer}`"
            :opacity="line.edge.opacity"
            text-anchor="middle"
            dominant-baseline="middle"
          ><textPath
            :href="`#${line.titlePath}`"
            startOffset="50%"
          >{{ line.edge.label }}</textPath></text>
          <text
            v-else-if="line.edge.label"
            class="plex__edge-label"
            :class="`plex__edge-label--${layer}`"
            :x="midpointOf(line.edge).x"
            :y="midpointOf(line.edge).y"
            :opacity="line.edge.opacity"
            text-anchor="middle"
            dominant-baseline="middle"
          >{{ line.edge.label }}</text>
        </template>
      </template>
    </g>

    <PlexNodeView
      v-for="node in frame.nodes"
      :key="node.id"
      :node="node"
      :standing="standingOf(node)"
      @activate="emit('activate', node.id)"
      @show="emit('show', node.id, $event)"
      @reach="emit('reach', node.id, $event)"
      @ask="emit('ask', node.id)"
      @menu="(at, from, opening) => emit('menu', node.id, at, from, opening)"
    >
      <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
    </PlexNodeView>

    <!-- The line under the hand, drawn after the boxes so it stands over
         them, and with it the title it carries. -->
    <g v-if="lifted.length" class="plex__lift" aria-hidden="true">
      <template v-for="line in lifted" :key="line.key">
        <path class="plex__edge" :d="line.d" :opacity="line.edge.opacity" />
        <template v-if="showEdgeLabels && line.edge.label">
          <template v-for="layer in TITLE_LAYERS" :key="layer">
            <text
              v-if="line.titlePath"
              class="plex__edge-label"
              :class="`plex__edge-label--${layer}`"
              :opacity="line.edge.opacity"
              text-anchor="middle"
              dominant-baseline="middle"
            ><textPath
              :href="`#${line.titlePath}`"
              startOffset="50%"
            >{{ line.edge.label }}</textPath></text>
            <text
              v-else
              class="plex__edge-label"
              :class="`plex__edge-label--${layer}`"
              :x="midpointOf(line.edge).x"
              :y="midpointOf(line.edge).y"
              :opacity="line.edge.opacity"
              text-anchor="middle"
              dominant-baseline="middle"
            >{{ line.edge.label }}</text>
          </template>
        </template>
      </template>
    </g>

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
  stroke: var(--numen-surface);
  stroke-width: var(--numen-edge-label-halo);
  stroke-linejoin: round;
  pointer-events: none;
}

/* The halo is stroke alone; the letters over it, fill alone. */
.plex__edge-label--halo {
  fill: none;
}

.plex__edge-label--letters {
  stroke: none;
}

/* Wide enough for a hand to land on, and unpainted. */
.plex__edge-hit {
  fill: none;
  stroke: transparent;
  stroke-width: 14px;
  pointer-events: stroke;
}

/* While the plex is moving, every line is on its way somewhere. */
[data-moving] .plex__edge-hit {
  pointer-events: none;
}

.plex__lift {
  pointer-events: none;
}

.plex__lift .plex__edge {
  stroke: color-mix(in oklab, var(--numen-edge), var(--numen-node-fg) 55%);
}

/* The title stands over the boxes here, on a halo as heavy as that asks for. */
.plex__lift .plex__edge-label--halo {
  stroke-width: 6px;
}

.plex__lift .plex__edge-label--letters {
  fill: var(--numen-node-fg);
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
