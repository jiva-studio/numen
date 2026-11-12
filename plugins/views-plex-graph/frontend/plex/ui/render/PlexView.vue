<script setup lang="ts">
import { computed, ref, useId, useTemplateRef } from 'vue'
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import PlexNodeView from './PlexNodeView.vue'
import PlexThread from './PlexThread.vue'
import { useEdgeLines } from './lines'
import type { PlexDrawnSlots, PlexViewEvents, PlexViewProps } from './props'
import { getDraggedShape, getPendingThread, getReachGhost, getReachThread } from './threads'
import type { GestureRole, PlacedNode } from '../../lib/node'
import { getRelationName } from '../../lib/relation'
import { DEFAULT_BEHAVIOR } from '../../model/behavior'
import { useNodeViews } from '../../model/useNodeViews'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PlexViewProps>(), {
  showEdgeLabels: true,
  showPills: true,
  canReach: true,
  relationName: getRelationName,
  behavior: () => DEFAULT_BEHAVIOR,
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

defineExpose({
  get svg() {
    return svg.value
  },
  focusNode: (id: string) => focusNode(id),
})

/* --------------------------------- State ---------------------------------- */
const svg = useTemplateRef<SVGSVGElement>('svg')

const { holdNode, focusNode } = useNodeViews()

const viewBox = computed(() => {
  const { width, height } = props.viewport
  return `${-width / 2} ${-height / 2} ${width} ${height}`
})

const uid = useId()

const { lines, raisedLines, resting, setOver } = useEdgeLines(() => props.frame, uid)

const restedOn = ref<string | null>(null)

const renderedNodes = computed(() => {
  const node = props.frame.nodes.find((each) => each.id === restedOn.value)
  if (!node) return props.frame.nodes
  return [...props.frame.nodes.filter((each) => each.id !== node.id), node]
})

const thread = computed(() =>
  getReachThread(props.frame, props.gestureFrom, props.gestureAt, props.curvature),
)

const pendingThread = computed(() =>
  getPendingThread(props.frame, {
    from: props.pending?.from ?? null,
    position: props.pending?.at ?? null,
    viewport: props.viewport,
    curvature: props.curvature,
  }),
)

const ghost = computed(() =>
  getReachGhost(props.gestureOutcome, props.gestureAt, {
    relationName: props.relationName,
    nodeSize: props.nodeSize,
  }),
)

const dragging = computed(() =>
  getDraggedShape(
    props.frame,
    { relation: props.dropRelation, to: props.draggedAt },
    { dropName: props.dropName, nodeSize: props.nodeSize, curvature: props.curvature },
  ),
)

/* -------------------------------- Handlers -------------------------------- */
const onSettle = (id: string, isResting: boolean) => {
  if (isResting) restedOn.value = id
  else if (restedOn.value === id) restedOn.value = null
}

/* -------------------------------- Helpers --------------------------------- */
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
    aria-label="Related notes"
  >
    <defs v-if="showEdgeLabels">
      <template v-for="line in lines" :key="`along:${line.key}`">
        <path v-for="title in line.titles" :id="title.path" :key="title.path" :d="title.line" />
      </template>
    </defs>

    <g aria-hidden="true">
      <PlexEdgeLine v-for="line in resting" :key="line.key" :line="line" />
    </g>

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
      :behavior="behavior"
      :show-pills="showPills"
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

    <g v-if="raisedLines.length" class="plex__raised" aria-hidden="true">
      <template v-for="line in raisedLines" :key="line.key">
        <PlexEdgeLine :line="line" is-raised />
        <PlexEdgeTitle v-if="showEdgeLabels" :line="line" is-raised />
      </template>
    </g>

    <PlexThread v-if="thread" class="plex__reach" :d="thread" :ghost="ghost" />

    <PlexThread v-if="pendingThread" class="plex__pending" :d="pendingThread" />

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
  touch-action: none;
  -webkit-touch-callout: none;
}

.plex__edge-hit {
  --edge-hit-stroke-width: 0.875rem;
  fill: none;
  stroke: transparent;
  stroke-width: var(--edge-hit-stroke-width);
  pointer-events: stroke;
}

[data-moving] .plex__edge-hit {
  pointer-events: none;
}

.plex__raised {
  pointer-events: none;
}

.plex__reach,
.plex__pending {
  pointer-events: none;
}

.plex__dragged {
  pointer-events: none;
}
</style>
