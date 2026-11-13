<script setup lang="ts">
/**
 * The plex: one focused node, with all other nodes placed by their relation to it.
 *
 * Takes a graph and emits an identifier when one is chosen. It does not
 * fetch data, does not interpret what an identifier addresses, and does not
 * compute who is related to whom. The caller answers `activate` with the next
 * graph, and the plex transitions to it.
 */
import { computed, useSlots, useTemplateRef, watch } from 'vue'
import PlexView from './render/PlexView.vue'
import type { PlexEvents, PlexProps, PlexSlots } from './props'
import { DEFAULT_BEHAVIOR, getBehavior, type PlexBehavior } from '../model/behavior'
import { useFreeSpace } from '../model/sizing/room'
import { usePlexTransition } from '../model/transition'
import { browserViewport } from '@numen/ui'
import { DEFAULT_FOCUSED_NAMES_OPACITY, DEFAULT_NAMES_OPACITY } from './namesOpacity'
import { formatCount, getRelationName } from '../lib/relation'
import { usePlexDrag } from '../model/interaction/drag'
import { usePlexGesture } from '../model/interaction/gesture'
import { getFrameStyle } from './frameStyle'
import { useNamesRooms } from '../model/useNamesRooms'
import type { PlexRelation } from '../../types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<PlexProps>(), {
  showEdgeLabels: true,
  showPills: true,
  lineWidth: 2,
  outlineWidth: 1,
  duration: 420,
  clock: () => DEFAULT_BEHAVIOR.clock,
  viewport: () => browserViewport,
  creatable: () => ['parent', 'child', 'jump'],
  dragThreshold: 8,
  expansionDelay: DEFAULT_BEHAVIOR.expansionDelay,
  reaching: () => DEFAULT_BEHAVIOR.reaching,
  showing: () => DEFAULT_BEHAVIOR.showing,
  relationName: getRelationName,
  pending: null,
  dragged: () => [],
  dropName: getRelationName,
  showNames: false,
  namesOpacity: DEFAULT_NAMES_OPACITY,
  focusedNamesOpacity: DEFAULT_FOCUSED_NAMES_OPACITY,
})

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<PlexEvents>()

defineExpose({
  get isMoving() {
    return isMoving.value
  },
  /** Restores keyboard focus to a node. */
  focusNode: (id: string) => view.value?.focusNode(id),
})

defineSlots<PlexSlots>()

/* --------------------------------- State ---------------------------------- */
const frameElement = useTemplateRef<HTMLElement>('frame')
/** The drawing, which a drag crossing the plex is measured against. */
const view = useTemplateRef<InstanceType<typeof PlexView>>('view')

const slots = useSlots()

const { room, options, measures, widen, hung } = useFreeSpace(
  frameElement,
  props,
  () => !!slots.icon,
)

/** How the nodes behave, handed down as one. */
const behavior = computed<PlexBehavior>(() => getBehavior(props))

const { getRoomOf, room: getRoom } = useNamesRooms(props, measures, options)

const { frame, isMoving } = usePlexTransition(
  () => props.graph,
  () => ({
    options: { ...props.options, viewport: room.value },
    placement: props.placement,
    measure: measures.value?.node,
    measureLabel: measures.value?.label,
    labelType: measures.value?.labelType,
    room: getRoom.value,
  }),
  {
    duration: () => props.duration,
    clock: props.clock,
  },
)

/**
 * A caller that allows no relation has disabled the gesture, so no handle is
 * shown. Pointer position over a node is handled by the node and does not
 * affect this.
 */
const canReach = computed(() => props.creatable.length > 0)

/**
 * Dragging out from a node.
 *
 * The plex reports the shape of the gesture: where it started, which way it
 * went and what it ended on. Whether a parent may be created, and what
 * creating one writes, is decided by the application.
 */
const gesture = usePlexGesture({
  frame: () => frame.value,
  options: () => options.value,
  relations: () => props.creatable,
  threshold: () => props.dragThreshold,
  commit: (drop) => {
    if (drop.kind === 'create') emit('create', drop.from, drop.relation)
    else emit('link', drop.from, drop.to, drop.relation)
  },
})

/**
 * An item dragged onto the plex from outside it.
 *
 * The plex determines which relation the drop evaluates to, measured from the
 * focus, and reports it. It does not inspect the dragged item.
 */
const dragging = usePlexDrag({
  getSurface: () => view.value?.svg ?? null,
  getDragged: () => props.dragged,
  getFrame: () => frame.value,
  getOptions: () => options.value,
  getViewport: () => room.value,
  getAllowedRelations: () => props.creatable,
  getThreshold: () => props.dragThreshold,
  commit: (dragged, relation) => emit('bring', dragged, relation),
})

/* --------------------------------- Hooks ---------------------------------- */
/**
 * The overflow, as `[relation, count]` pairs. The renderer of the plex supplies
 * the labels through the `overflow` slot.
 */
const overflow = computed(() =>
  (Object.entries(frame.value.overflow) as [PlexRelation, number][]).filter(
    ([, count]) => count > 0,
  ),
)

const frameStyle = computed(() => getFrameStyle(props, options.value))

/**
 * A graph is drawn from the node it is seen from, so every node walks
 * to a new relation when another one arrives. A menu is anchored where one of them
 * was.
 */
watch(
  () => props.graph,
  () => emit('dismiss'),
)
</script>

<template>
  <div
    ref="frame"
    class="plex-frame numen"
    :data-moving="isMoving || undefined"
    :style="frameStyle"
  >
    <PlexView
      ref="view"
      :frame="frame"
      :viewport="room"
      :node-size="options.nodeSize"
      :curvature="options.routing.curvature"
      :show-edge-labels="showEdgeLabels"
      :show-pills="showPills"
      :names-opacity="namesOpacity"
      :focused-names-opacity="focusedNamesOpacity"
      :can-reach="canReach"
      :relation-name="relationName"
      :widen="widen"
      :hung="hung"
      :room-of="getRoomOf"
      :behavior="behavior"
      :gesture-from="gesture.from.value"
      :gesture-at="gesture.at.value"
      :gesture-outcome="gesture.outcome.value"
      :pending="pending"
      :dragged-at="dragging.at.value"
      :drop-relation="dragging.relation.value"
      :drop-name="dropName"
      @activate="emit('activate', $event)"
      @show="(id, showing) => emit('show', id, showing)"
      @reach="gesture.begin"
      @ask="gesture.ask"
      @menu="(id, at, trigger) => emit('menu', id, at, trigger)"
      @enter="(id, heading) => emit('enter', id, heading)"
      @choose-name="emit('chooseName', $event)"
      @edge-menu="(pair, at) => emit('edge-menu', pair, at)"
    >
      <template v-if="$slots.icon" #icon="{ node }"><slot name="icon" :node="node" /></template>
    </PlexView>
    <div v-if="overflow.length" class="plex-frame__overflow" role="status">
      <slot name="overflow" :overflow="overflow">
        {{ overflow.map(([relation, count]) => formatCount(relation, count)).join(', ') }} not shown
      </slot>
    </div>
  </div>
</template>

<style scoped>
.plex-frame {
  position: relative;
  inline-size: 100%;
  block-size: 100%;
  background: var(--numen-surface);
}

.plex-frame__overflow {
  position: absolute;
  user-select: none;
  -webkit-user-select: none;
  inset-block-end: var(--numen-inset);
  inset-inline-start: var(--numen-inset-wide);
  color: var(--numen-edge-label);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-edge-label-size);
}
</style>
