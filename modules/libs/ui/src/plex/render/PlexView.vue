<script setup lang="ts">
/**
 * The drawing, and nothing else. Every number here came from `arrange/`.
 *
 * What it draws itself is the picture between the nodes: the window, the
 * edges, and the gesture crossing them. A node draws itself, and is told the
 * two things about it that only the whole picture knows.
 */
import { computed, ref, useId, useTemplateRef, watch } from 'vue'
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import PlexNodeView from './PlexNodeView.vue'
import type { EdgeLine } from './lines'
import { edgeKey } from '../edge'
import type { PlexFrame } from '../frame'
import { ghostNode, handleIn, type GestureRole, type PlacedNode, type Point } from '../node'
import { seatWord, type PlexRelatedSeat } from '../seat'
import { DWELL, type Widened } from '../dwell'
import { byHandle, type ReachStrategy } from '../reaching'
import { byDoubleClick, type PlexShowing, type ShowStrategy } from '../showing'
import type { HungParts } from '../inside'
import { browserClock, type Clock } from '../transition'
import {
  arrowTransformOf,
  pathOf,
  readingPathOf,
  threadOf,
  type Drop,
} from '../arrange'
import type { MenuOpening } from '../../menu/item'

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
    /**
     * The box a node widens to while the attention rests on it, and nothing
     * for a node with no more of its title to show. Text is measured where the
     * plex is drawn, so this arrives already worked out.
     */
    widen?: ((node: PlacedNode) => Widened | null) | undefined
    /**
     * The parts a node hangs under its box while the attention rests on it,
     * and nothing for a node with none. Which parts a node holds is the
     * picture's to work out.
     */
    hung?: ((node: PlacedNode) => HungParts | null) | undefined
    /** How long the attention rests on a box before it widens. Milliseconds. */
    dwell?: number
    /** How a node offers to be reached out of. The handle by default. */
    reaching?: ReachStrategy
    /** How a node is asked for on its own. The second click by default. */
    showing?: ShowStrategy
    /** The clock a box opens on. Browser by default; a test hands in its own. */
    clock?: Clock
    /** A gesture in progress: where it started, where it is, what it means. */
    gestureFrom?: string | null
    gestureAt?: Point | null
    gestureOutcome?: Drop | null
    /**
     * Something carried over the picture from outside it: where the pointer
     * is, and the seat letting go there comes to. Both, or the picture draws
     * none of it.
     */
    carriedAt?: Point | null
    carriedSeat?: PlexRelatedSeat | null
    /**
     * What to call what letting go with something carried in would do, for the
     * one place it is written into the picture. English by default.
     */
    carriedName?: (seat: PlexRelatedSeat) => string
  }>(),
  {
    showEdgeLabels: true,
    mayReach: true,
    seatName: seatWord,
    dwell: DWELL,
    reaching: () => byHandle,
    showing: () => byDoubleClick,
    clock: () => browserClock,
    gestureFrom: null,
    gestureAt: null,
    gestureOutcome: null,
    carriedAt: null,
    carriedSeat: null,
    carriedName: seatWord,
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
  /** A menu was asked for on a node: which, where, and by what. */
  (event: 'menu', id: string, at: Point, opening: MenuOpening): void
  /** A part of a node was chosen. Both identifiers are the caller's. */
  (event: 'enter', id: string, part: string): void
}>()

const svg = useTemplateRef<SVGSVGElement>('svg')

/** The boxes as they are drawn, so the keyboard can be put back on one. */
const views = new Map<string, { focus: () => void }>()

const holdNode = (id: string, view: unknown): void => {
  if (view) views.set(id, view as { focus: () => void })
  else views.delete(id)
}

defineExpose({ svg, focusNode: (id: string) => views.get(id)?.focus() })

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

/** Two plexes on one page each name their own paths. */
const uid = useId()

/**
 * Every edge with what the drawing asks of it. A title is always set along the
 * line it belongs to, in the words the arrangement cut for it and at the place
 * along it the arrangement chose.
 */
const lines = computed<readonly EdgeLine[]>(() =>
  props.frame.edges.map((edge, at) => ({
    edge,
    key: `${edge.from}->${edge.to}`,
    pair: edgeKey(edge),
    d: pathOf(edge),
    arrow: edge.arrowhead ? arrowTransformOf(edge.arrowhead) : null,
    titlePath: edge.words ? `${uid}-title-${at}` : null,
    titleLine: readingPathOf(edge),
    titleAt: `${100 * edge.wordsAt}%`,
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
 * What each node is to the gesture. Only the node it left from keeps a handle
 * while one is running: the hand is somewhere else entirely, and a second
 * handle under it would offer to start a gesture already under way.
 */
const roleOf = (node: PlacedNode): GestureRole => {
  const outcome = props.gestureOutcome
  if (outcome?.kind === 'link' && outcome.to === node.id) return 'target'
  if (props.gestureFrom === node.id) return 'source'
  return props.mayReach && props.gestureFrom === null ? 'open' : 'closed'
}

/** The node the attention has settled on, as that node reports it. */
const restedOn = ref<string | null>(null)

const rest = (id: string, settled: boolean) => {
  if (settled) restedOn.value = id
  else if (restedOn.value === id) restedOn.value = null
}

/**
 * The boxes in the order they are drawn, the one being rested on last. It is
 * drawn wider than it was placed, and what it covers is drawn under it.
 */
const drawn = computed(() => {
  const node = props.frame.nodes.find((each) => each.id === restedOn.value)
  if (!node) return props.frame.nodes
  return [...props.frame.nodes.filter((each) => each.id !== node.id), node]
})

/** The line a gesture drags behind it, from the handle to the pointer. */
const thread = computed(() => {
  const source = props.frame.nodes.find((node) => node.id === props.gestureFrom)
  const to = props.gestureAt
  if (!source || !to) return null
  const offset = handleIn(source)
  return threadOf({ x: source.x + offset.x, y: source.y + offset.y }, to)
})

/**
 * Something carried over the picture: the line from the focus to the pointer,
 * and the shape letting go would leave there.
 *
 * Drawn only where letting go comes to a seat, so what the reader sees and
 * what the gesture answers are the one thing. The shape is a box like any
 * other, saying which seat it would take in whatever words it was given.
 */
const carrying = computed(() => {
  const seat = props.carriedSeat
  const to = props.carriedAt
  const focus = props.frame.nodes.find((node) => node.seat === 'focus')
  if (!seat || !to || !focus) return null

  const ghost = ghostNode('carried', props.carriedName(seat), seat, to, props.nodeSize)
  return { thread: threadOf(focus, to), ghost }
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
  return ghostNode('ghost', props.seatName(outcome.seat), outcome.seat, to, props.nodeSize)
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
      <PlexEdgeLine v-for="line in resting" :key="line.key" :line="line" />
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
      <PlexEdgeTitle v-for="line in resting" :key="`title:${line.key}`" :line="line" />
    </g>

    <PlexNodeView
      v-for="node in drawn"
      :key="node.id"
      :node="node"
      :gesture-role="roleOf(node)"
      :wide="widen?.(node) ?? null"
      :hung="hung?.(node) ?? null"
      :dwell="dwell"
      :reaching="reaching"
      :showing="showing"
      :clock="clock"
      @activate="emit('activate', node.id)"
      @show="emit('show', node.id, $event)"
      @reach="emit('reach', node.id, $event)"
      @ask="emit('ask', node.id)"
      :ref="(view) => holdNode(node.id, view)"
      @menu="(at, opening) => emit('menu', node.id, at, opening)"
      @enter="(part) => emit('enter', node.id, part)"
      @rest="rest(node.id, $event)"
    >
      <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
    </PlexNodeView>

    <!-- The line under the hand, drawn after the boxes so it stands over
         them, and with it the title it carries. -->
    <g v-if="lifted.length" class="plex__lift" aria-hidden="true">
      <template v-for="line in lifted" :key="line.key">
        <PlexEdgeLine :line="line" lifted />
        <PlexEdgeTitle v-if="showEdgeLabels" :line="line" lifted />
      </template>
    </g>

    <!-- The gesture itself, drawn over everything it may land on. -->
    <g v-if="thread" class="plex__reach">
      <path class="plex__thread" :d="thread" aria-hidden="true" />
      <PlexNodeView v-if="ghost" :node="ghost" gesture-role="ghost" />
    </g>

    <!-- Something carried in from outside, drawn over everything it crosses. -->
    <g v-if="carrying" class="plex__carried">
      <path class="plex__thread" :d="carrying.thread" aria-hidden="true" />
      <PlexNodeView :node="carrying.ghost" gesture-role="ghost" />
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
  -webkit-user-select: none;
  /* Every touch on the picture belongs to the picture, and a finger that
     travels is carrying something across it. */
  touch-action: none;
  -webkit-touch-callout: none;
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

.plex__reach {
  pointer-events: none;
}

/* What is carried across the picture catches nothing on its way over. */
.plex__carried {
  pointer-events: none;
}

.plex__thread {
  fill: none;
  stroke: var(--numen-ring);
  stroke-width: var(--numen-edge-width);
  stroke-dasharray: var(--numen-thread-dash);
}
</style>
