<script setup lang="ts">
import { computed, ref, useId, useTemplateRef } from 'vue'
import PlexEdgeLayer from './PlexEdgeLayer.vue'
import PlexNodeView from './PlexNodeView.vue'
import PlexNodeNames from './PlexNodeNames.vue'
import PlexThreads from './PlexThreads.vue'
import { useEdgeLines } from './lines'
import { listNamedNodes } from './named'
import type { PlexDrawnSlots, PlexViewEvents, PlexViewProps } from './props'
import type { GestureRole, PlacedNode } from '../../lib/node'
import { getRelationName } from '../../lib/relation'
import { DEFAULT_BEHAVIOR } from '../../model/behavior'
import { useAttention } from '../../model/useAttention'
import { useNodeHandlers } from '../../model/useNodeHandlers'
import { useNodeViews } from '../../model/useNodeViews'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PlexViewProps>(), {
  showEdgeLabels: true,
  showPills: true,
  highlightLinks: false,
  canReach: true,
  relationName: getRelationName,
  behavior: () => DEFAULT_BEHAVIOR,
  gesture: undefined,
  dragging: undefined,
  pending: null,
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

const restedOn = ref<string | null>(null)

const { attention, attend } = useAttention(() => props.frame)

const { lines, isRaised, setOver } = useEdgeLines(props, uid, attention)

const getNodeHandlers = useNodeHandlers({
  activate: (id) => emit('activate', id),
  show: (id, showing) => emit('show', id, showing),
  reach: (id, pointer) => emit('reach', id, pointer),
  ask: (id) => emit('ask', id),
  menu: (id, at, trigger) => emit('menu', id, at, trigger),
  enter: (id, heading) => emit('enter', id, heading),
  settle: (id, isResting) => onSettle(id, isResting),
  attend: (id, isAttended) => attend('node', id, isAttended),
})

const namedNodes = computed(() => listNamedNodes(props.frame.nodes, props.roomOf))

const renderedNodes = computed(() => {
  const node = props.frame.nodes.find((each) => each.id === restedOn.value)
  if (!node) return props.frame.nodes
  return [...props.frame.nodes.filter((each) => each.id !== node.id), node]
})

/** What each note is settled to: its width and its headings, kept as they are until the frame or the sizes change. */
const wides = computed(
  () => new Map(props.frame.nodes.map((node) => [node.id, props.widen?.(node) ?? null])),
)

const hungs = computed(
  () => new Map(props.frame.nodes.map((node) => [node.id, props.hung?.(node) ?? null])),
)

/** Stays the same map while no role changes, so a pointer moving through a gesture does not redraw the notes. */
const roles = computed<Map<string, GestureRole>>((previous) => {
  const next = new Map(props.frame.nodes.map((node) => [node.id, getRole(node)]))
  const isSame =
    previous !== undefined &&
    previous.size === next.size &&
    [...next].every(([id, role]) => previous.get(id) === role)
  return isSame ? previous : next
})

/* -------------------------------- Handlers -------------------------------- */
const onSettle = (id: string, isResting: boolean) => {
  if (isResting) restedOn.value = id
  else if (restedOn.value === id) restedOn.value = null
}

const onChooseName = (id: string) => {
  attention.names = null
  emit('choose-name', id)
}

/* -------------------------------- Helpers --------------------------------- */
const getRole = (node: PlacedNode): GestureRole => {
  const outcome = props.gesture?.outcome.value
  const from = props.gesture?.from.value ?? null
  if (outcome?.kind === 'link' && outcome.to === node.id) return 'target'
  if (from === node.id) return 'source'
  return props.canReach && from === null ? 'open' : 'closed'
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

    <PlexEdgeLayer
      layer="resting"
      :lines="lines"
      :is-raised="isRaised"
      :show-edge-labels="showEdgeLabels"
    />

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

    <PlexEdgeLayer
      v-if="showEdgeLabels"
      layer="titles"
      :lines="lines"
      :is-raised="isRaised"
      :show-edge-labels="showEdgeLabels"
    />

    <!-- Under every node, so the headings that open under one are over the names of all. -->
    <g v-if="roomOf">
      <PlexNodeNames
        v-for="named in namedNodes"
        :key="`names:${named.node.id}`"
        :node="named.node"
        :room="named.room"
        :attention="attention"
        :resting-opacity="namesOpacity"
        :focused-opacity="focusedNamesOpacity"
        @attend="attend('names', named.node.id, $event)"
        @choose="onChooseName"
      />
    </g>

    <!-- Over the names and the other lines, and under every node, so the headings that open under a node cover them. -->
    <PlexEdgeLayer
      layer="raised"
      :lines="lines"
      :is-raised="isRaised"
      :show-edge-labels="showEdgeLabels"
    />

    <g v-for="node in renderedNodes" :key="node.id" class="plex__stack">
      <PlexNodeView
        :ref="holdNode(node.id)"
        :node="node"
        :gesture-role="roles.get(node.id) ?? 'closed'"
        :wide="wides.get(node.id) ?? null"
        :hung="hungs.get(node.id) ?? null"
        :behavior="behavior"
        :show-pills="showPills"
        :rest-height="nodeSize.height"
        :attention="attention"
        v-on="getNodeHandlers(node.id)"
      >
        <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
      </PlexNodeView>
    </g>

    <PlexThreads
      :frame="frame"
      :viewport="viewport"
      :ghost-size="ghostSize"
      :relation-name="relationName"
      :gesture="gesture"
      :dragging="dragging"
      :pending="pending"
      :drop-name="dropName"
      :curvature="curvature"
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
</style>
