<script setup lang="ts">
/**
 * The plex: one node in focus, everything else placed by its relation.
 *
 * Takes a neighbourhood, gives back an identifier when one is chosen. It does
 * not fetch, does not know what an identifier addresses, and does not work out
 * who is related to whom. Answer `activate` with the next neighbourhood and it
 * travels there by itself.
 */
import { computed, toRef, useSlots, useTemplateRef, watch } from 'vue'
import PlexView from './render/PlexView.vue'
import type { PlexEvents, PlexProps, PlexSlots } from './props'
import { DEFAULT_BEHAVIOUR, type PlexBehaviour } from '../model/behaviour'
import { useRoom } from '../model/room'
import { usePlexTransition } from '../model/transition'
import { browserViewport } from '@/shared/lib/viewport'
import { formatCount, getRelationName, type PlexRelation } from '../lib/relation'
import { usePlexDrag } from '../model/drag'
import { usePlexGesture } from '../model/gesture'

const props = withDefaults(defineProps<PlexProps>(), {
  showEdgeLabels: true,
  duration: 420,
  clock: () => DEFAULT_BEHAVIOUR.clock,
  viewport: () => browserViewport,
  creatable: () => ['parent', 'child', 'jump'],
  dragThreshold: 8,
  expansionDelay: DEFAULT_BEHAVIOUR.expansionDelay,
  reaching: () => DEFAULT_BEHAVIOUR.reaching,
  showing: () => DEFAULT_BEHAVIOUR.showing,
  relationName: getRelationName,
  pending: null,
  dragged: () => [],
  dropName: getRelationName,
})

const emit = defineEmits<PlexEvents>()

defineSlots<PlexSlots>()

const frameElement = useTemplateRef<HTMLElement>('frame')
/** The drawing, which a drag crossing the plex is measured against. */
const view = useTemplateRef<InstanceType<typeof PlexView>>('view')

const slots = useSlots()

const { room, options, measures, widen, hung } = useRoom(frameElement, props, () => !!slots.icon)

/** How the nodes behave, handed down as one. */
const behaviour = computed<PlexBehaviour>(() => ({
  expansionDelay: props.expansionDelay,
  reaching: props.reaching,
  showing: props.showing,
  clock: props.clock,
}))

const { frame, isMoving } = usePlexTransition(
  () => props.neighbourhood,
  () => ({
    options: { ...props.options, viewport: room.value },
    placement: props.placement,
    measure: measures.value?.node,
    measureLabel: measures.value?.label,
    labelDepth: measures.value?.labelDepth,
  }),
  {
    duration: () => props.duration,
    clock: props.clock,
  },
)

/**
 * A caller that allows no relation at all has turned the gesture off, and a handle
 * that can come to nothing is a lie. Where a pointer happens to be is the
 * node's own affair, and never reaches this far.
 */
const canReach = computed(() => props.creatable.length > 0)

/**
 * Reaching out from a node.
 *
 * The plex works out the shape of it — where it started, which way it went,
 * what it landed on — and says so. Whether a parent may be made, and what
 * making one writes, is the application's.
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
 * Something dragged across the picture from outside it.
 *
 * The plex works out which relation letting go evaluates to, measured from the focus,
 * and says so. What is being dragged it never looks at.
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

/**
 * A neighbourhood is drawn from the node it is seen from, so every node walks
 * to a new relation when another one arrives. A menu is anchored where one of them
 * was.
 */
watch(
  () => props.neighbourhood,
  () => emit('dismiss'),
)

/**
 * What did not fit, as `[relation, count]` pairs. The words belong to whoever
 * renders the plex, through the `overflow` slot.
 */
const overflow = computed(() =>
  (Object.entries(frame.value.overflow) as [PlexRelation, number][]).filter(
    ([, count]) => count > 0,
  ),
)

const frameStyle = computed(() => ({
  '--numen-plex-move': `${props.duration}ms`,
}))

defineExpose({
  isMoving: toRef(isMoving),
  /** The keyboard put back on a node by whoever took it away. */
  focusNode: (id: string) => view.value?.focusNode(id),
})
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
      :show-edge-labels="showEdgeLabels"
      :can-reach="canReach"
      :relation-name="relationName"
      :widen="widen"
      :hung="hung"
      :behaviour="behaviour"
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
