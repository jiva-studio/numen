<script setup lang="ts">
import { computed, provide, useId, useTemplateRef } from 'vue'
import PlexEdgeLayer from './PlexEdgeLayer.vue'
import PlexNodeView from './PlexNodeView.vue'
import PlexNodeWidgets from './PlexNodeWidgets.vue'
import PlexThreads from './PlexThreads.vue'
import { LIFT_LAYER, NODE_SIZES, OVER_LAYER } from './layer'
import { useEdgeLines } from './lines'
import type { PlexDrawnSlots, PlexViewEvents, PlexViewProps } from './props'
import type { GestureRole, PlacedNode } from '../../lib/node'
import { getRelationName } from '../../lib/relation'
import type { PlexWidget } from '../../../types'
import { DEFAULT_BEHAVIOR } from '../../model/behavior'
import { useAttention } from '../../model/useAttention'
import { useNodeHandlers } from '../../model/useNodeHandlers'
import { useNodeViews } from '../../model/useNodeViews'
import { useNodeSizes } from '../../model/useNodeSizes'

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

const lift = useTemplateRef<SVGGElement>('lift')
provide(LIFT_LAYER, lift)
const over = useTemplateRef<SVGGElement>('over')
provide(OVER_LAYER, over)

const viewBox = computed(() => {
  const { width, height } = props.viewport
  return `${-width / 2} ${-height / 2} ${width} ${height}`
})

const uid = useId()

const { attention, setAttention } = useAttention(() => props.frame)

const { lines, isRaised, setOver, getLabel } = useEdgeLines(props, uid, attention)

const getNodeHandlers = useNodeHandlers(
  {
    activate: (id) => emit('activate', id),
    show: (id, showing) => emit('show', id, showing),
    reach: (id, pointer) => emit('reach', id, pointer),
    ask: (id) => emit('ask', id),
    openMenu: (id, at, trigger) => emit('menu', id, at, trigger),
    setAttention,
  },
  () => props.frame.nodes.map((node) => node.id),
)

const NO_WIDGETS: readonly PlexWidget[] = []
const overlayEntries = computed(() => props.widgetsOver ?? NO_WIDGETS)

const stacked = computed(() => [...props.frame.nodes, ...(props.waiting ?? [])])
const waitingIds = computed(() => new Set(props.waiting?.map((node) => node.id)))

const sizes = useNodeSizes(() => ({
  nodes: props.frame.nodes,
  widen: props.widen,
}))
provide(NODE_SIZES, sizes)

/** Stays the same map while no role changes, so a pointer moving through a gesture does not redraw the notes. */
const roles = computed<Map<string, GestureRole>>((previous) => {
  const next = new Map(props.frame.nodes.map((node) => [node.id, getRole(node)]))
  const isSame =
    previous !== undefined &&
    previous.size === next.size &&
    [...next].every(([id, role]) => previous.get(id) === role)
  return isSame ? previous : next
})

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
        role="button"
        :aria-label="getLabel(line)"
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

    <!-- Over the other lines, and under every node. -->
    <PlexEdgeLayer
      layer="raised"
      :lines="lines"
      :is-raised="isRaised"
      :show-edge-labels="showEdgeLabels"
    />

    <g
      v-for="node in stacked"
      :key="node.id"
      class="plex__stack"
      :class="{ 'plex__stack--waiting': waitingIds.has(node.id) }"
    >
      <PlexNodeView
        v-if="!waitingIds.has(node.id)"
        :ref="holdNode(node.id)"
        :node="node"
        :gesture-role="roles.get(node.id) ?? 'closed'"
        :behavior="behavior"
        :show-pills="showPills"
        :rest-height="nodeSize.height"
        :viewport-width="viewport.width"
        :attention="attention"
        :overlay="overlayEntries"
        :path="getPath?.(node.id) ?? ''"
        :viewport="viewport"
        v-on="getNodeHandlers(node.id)"
      >
        <template v-if="$slots.icon" #icon><slot name="icon" :node="node" /></template>
      </PlexNodeView>
      <PlexNodeWidgets
        v-for="side in ['above', 'below'] as const"
        :key="side"
        :node="node"
        :side="side"
        :entries="(side === 'above' ? widgetsAbove : widgetsBelow) ?? []"
        :path="getPath?.(node.id) ?? ''"
        :width="nodeSize.width"
        :clock="clock"
        @widgets="setWidgets?.(node.id, side, $event)"
        @ready="setReady?.(node.id, side, $event)"
      />
    </g>

    <!-- Over every note, so a note that is open covers its neighbors without moving in the document. -->
    <g ref="lift" class="plex__lift" />

    <!-- Over every note and every open note, so a view across a note may reach over its neighbors. -->
    <g ref="over" class="plex__over-layer" />

    <PlexThreads
      :frame="frame"
      :viewport="viewport"
      :show-pills="showPills"
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

.plex__stack--waiting {
  visibility: hidden;
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
