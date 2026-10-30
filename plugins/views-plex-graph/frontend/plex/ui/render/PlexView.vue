<script setup lang="ts">
/**
 * The drawing layer. Every number here is computed in `arrange/`.
 *
 * It draws the viewport, the edges between nodes, and the gesture across them.
 * Each node draws itself and receives the two values that only the whole
 * drawing can determine.
 */
import { computed, ref, useId, useTemplateRef } from 'vue'
import { createRefRegistry } from '@numen/ui'
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import PlexNodeView from './PlexNodeView.vue'
import PlexThread from './PlexThread.vue'
import { useEdgeLines } from './lines'
import type { PlexDrawnSlots, PlexViewEvents, PlexViewProps } from './props'
import { getDraggedShape, getPendingThread, getReachGhost, getReachThread } from './threads'
import type { GestureRole, PlacedNode } from '../../lib/node'
import { getRelationName } from '../../lib/relation'
import { DEFAULT_BEHAVIOUR } from '../../model/behaviour'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PlexViewProps>(), {
  showEdgeLabels: true,
  canReach: true,
  relationName: getRelationName,
  behaviour: () => DEFAULT_BEHAVIOUR,
  gestureFrom: null,
  gestureAt: null,
  gestureOutcome: null,
  pending: null,
  draggedAt: null,
  dropRelation: null,
  dropName: getRelationName,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<PlexViewEvents>()

defineSlots<PlexDrawnSlots>()

/* --------------------------------- State ---------------------------------- */
const svg = useTemplateRef<SVGSVGElement>('svg')

/** The boxes as they are drawn, so the keyboard can be put back on one. */
const views = new Map<string, { focus: () => void }>()
const { getRef: holdNode } = createRefRegistry<string>((id, view) => {
  if (view) views.set(id, view as { focus: () => void })
  else views.delete(id)
})

defineExpose({ svg, focusNode: (id: string) => views.get(id)?.focus() })

/**
 * One plex unit is one pixel, origin at the middle of the window.
 *
 * The viewBox is a fixed window on the drawing, centred on the focus, so a box
 * and a letter are one size however many neighbours arrive. A neighbourhood
 * too wide is clipped, and the per-relation limits and the overflow count say how
 * much.
 */
const viewBox = computed(() => {
  const { width, height } = props.viewport
  return `${-width / 2} ${-height / 2} ${width} ${height}`
})

/** Two plexes on one page each name their own paths. */
const uid = useId()

const { lines, lifted, resting, setOver } = useEdgeLines(() => props.frame, uid)

/** The node the attention has settled on, as that node reports it. */
const restedOn = ref<string | null>(null)

/**
 * The boxes in the order they are drawn, the one being rested on last. It is
 */
const renderedNodes = computed(() => {
  const node = props.frame.nodes.find((each) => each.id === restedOn.value)
  if (!node) return props.frame.nodes
  return [...props.frame.nodes.filter((each) => each.id !== node.id), node]
})

/** The line a gesture drags behind it, from the handle to the pointer. */
const thread = computed(() => getReachThread(props.frame, props.gestureFrom, props.gestureAt))

/** The line from a node to a point awaiting a node of its own. */
const pendingThread = computed(() =>
  getPendingThread(props.frame, {
    from: props.pending?.from ?? null,
    position: props.pending?.at ?? null,
    viewport: props.viewport,
  }),
)

/** The node a gesture would make, drawn where it would appear. */
const ghost = computed(() =>
  getReachGhost(props.gestureOutcome, props.gestureAt, {
    relationName: props.relationName,
    nodeSize: props.nodeSize,
  }),
)

/** What is dragged in from outside, and the shape letting go would leave. */
const dragging = computed(() =>
  getDraggedShape(
    props.frame,
    { relation: props.dropRelation, to: props.draggedAt },
    { dropName: props.dropName, nodeSize: props.nodeSize },
  ),
)

/* -------------------------------- Handlers -------------------------------- */
const onSettle = (id: string, isResting: boolean) => {
  if (isResting) restedOn.value = id
  else if (restedOn.value === id) restedOn.value = null
}

/* -------------------------------- Helpers --------------------------------- */
/**
 * The role of each node in the current gesture. While a gesture runs, only the
 * node it started from keeps a handle, because a second handle would offer to
 * start a gesture while one is already in progress.
 */
const getRole = (node: PlacedNode): GestureRole => {
  const outcome = props.gestureOutcome
  if (outcome?.kind === 'link' && outcome.to === node.id) return 'target'
  if (props.gestureFrom === node.id) return 'source'
  return props.canReach && props.gestureFrom === null ? 'open' : 'closed'
}
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
    <g>
      <path
        v-for="line in lines"
        :key="`reach:${line.key}`"
        class="plex__edge-hit"
        :d="line.d"
        @pointerenter="setOver(line.pair)"
        @pointerleave="setOver(null)"
        @click.stop="emit('edge-menu', line.pair, { x: $event.clientX, y: $event.clientY })"
        @contextmenu.prevent.stop="
          emit('edge-menu', line.pair, { x: $event.clientX, y: $event.clientY })
        "
      />
    </g>

    <g v-if="showEdgeLabels" aria-hidden="true">
      <PlexEdgeTitle v-for="line in resting" :key="`title:${line.key}`" :line="line" />
    </g>

    <PlexNodeView
      v-for="node in renderedNodes"
      :key="node.id"
      :ref="holdNode(node.id)"
      :node="node"
      :gesture-role="getRole(node)"
      :wide="widen?.(node) ?? null"
      :hung="hung?.(node) ?? null"
      :behaviour="behaviour"
      @activate="emit('activate', node.id)"
      @show="emit('show', node.id, $event)"
      @reach="emit('reach', node.id, $event)"
      @ask="emit('ask', node.id)"
      @menu="(at, trigger) => emit('menu', node.id, at, trigger)"
      @enter="(heading) => emit('enter', node.id, heading)"
      @settle="onSettle(node.id, $event)"
    >
      <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
    </PlexNodeView>

    <!-- The line under the hand, drawn after the boxes so it stands over
         them, and with it the title it carries. -->
    <g v-if="lifted.length" class="plex__lift" aria-hidden="true">
      <template v-for="line in lifted" :key="line.key">
        <PlexEdgeLine :line="line" is-raised />
        <PlexEdgeTitle v-if="showEdgeLabels" :line="line" is-raised />
      </template>
    </g>

    <!-- The gesture itself, drawn over everything it may land on. -->
    <PlexThread v-if="thread" class="plex__reach" :d="thread" :ghost="ghost" />

    <PlexThread v-if="pendingThread" class="plex__pending" :d="pendingThread" />

    <!-- Something dragged in from outside, drawn over everything it crosses. -->
    <PlexThread
      v-if="dragging"
      class="plex__dragged"
      :d="dragging.thread"
      :ghost="dragging.ghost"
    />
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
  /* Every touch on the drawing is handled by the drawing, and a finger that
     travels is dragging something across it. */
  touch-action: none;
  -webkit-touch-callout: none;
}

/* Wide enough to be a pointer target, and unpainted. */
.plex__edge-hit {
  fill: none;
  stroke: transparent;
  stroke-width: 14px;
  pointer-events: stroke;
}

/* While the plex animates, edges do not receive pointer events. */
[data-moving] .plex__edge-hit {
  pointer-events: none;
}

.plex__lift {
  pointer-events: none;
}

.plex__reach,
.plex__pending {
  pointer-events: none;
}

/* The dragged element does not receive pointer events. */
.plex__dragged {
  pointer-events: none;
}
</style>
