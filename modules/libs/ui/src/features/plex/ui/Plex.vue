<script setup lang="ts">
/**
 * The plex: one node in focus, everything else placed by its seat.
 *
 * Takes a neighbourhood, gives back an identifier when one is chosen. It does
 * not fetch, does not know what an identifier addresses, and does not work out
 * who is related to whom. Answer `activate` with the next neighbourhood and it
 * travels there by itself.
 */
import { computed, toRef, useSlots, useTemplateRef, watch } from 'vue'
import PlexView from './render/PlexView.vue'
import type { PlexEvents, PlexProps, PlexSlots } from './props'
import { DWELL } from '../model/dwell'
import { byHandle } from '../model/reaching'
import { byDoubleClick } from '../model/showing'
import { useRoom } from '../model/room'
import { usePlexTransition, browserClock } from '../model/transition'
import { browserViewport } from '@/shared/lib/viewport'
import { countOf, seatWord, type PlexRelatedSeat } from '../lib/seat'
import { usePlexDrag } from '../model/drag'
import { usePlexGesture } from '../model/gesture'

const props = withDefaults(defineProps<PlexProps>(), {
  showEdgeLabels: true,
  duration: 420,
  clock: () => browserClock,
  viewport: () => browserViewport,
  creatable: () => ['parent', 'child', 'jump'],
  dragThreshold: 8,
  dwell: DWELL,
  reaching: () => byHandle,
  showing: () => byDoubleClick,
  seatName: seatWord,
  dragged: () => [],
  dropName: seatWord,
})

const emit = defineEmits<PlexEvents>()

defineSlots<PlexSlots>()

const frameElement = useTemplateRef<HTMLElement>('frame')
/** The drawing, which a drag crossing the plex is measured against. */
const view = useTemplateRef<InstanceType<typeof PlexView>>('view')

const slots = useSlots()

const { room, options, measures, widen, hung } = useRoom(frameElement, props, () => !!slots.icon)

const { frame, isMoving } = usePlexTransition(
  () => props.neighbourhood,
  () => ({
    options: { ...props.options, viewport: room.value },
    placement: props.placement,
    measure: measures.value?.node,
    measureLabel: measures.value?.label,
    labelDepth: measures.value?.labelDepth,
  }),
  () => props.duration,
  props.clock,
)

/**
 * A caller that allows no seat at all has turned the gesture off, and a handle
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
const gesture = usePlexGesture(
  () => frame.value,
  () => options.value,
  () => props.creatable,
  () => props.dragThreshold,
  (drop) => {
    if (drop.kind === 'create') emit('create', drop.from, drop.seat)
    else emit('link', drop.from, drop.to, drop.seat)
  },
)

/**
 * Something dragged across the picture from outside it.
 *
 * The plex works out which seat letting go comes to, measured from the focus,
 * and says so. What is being dragged it never looks at.
 */
const dragging = usePlexDrag({
  getSurface: () => view.value?.svg ?? null,
  getDragged: () => props.dragged,
  getFrame: () => frame.value,
  getOptions: () => options.value,
  getViewport: () => room.value,
  getAllowedSeats: () => props.creatable,
  getThreshold: () => props.dragThreshold,
  settle: (dragged, seat) => emit('bring', dragged, seat),
})

/**
 * A neighbourhood is drawn from the node it is seen from, so every node walks
 * to a new seat when another one arrives. A menu is anchored where one of them
 * was.
 */
watch(
  () => props.neighbourhood,
  () => emit('dismiss'),
)

/**
 * What did not fit, as `[seat, count]` pairs rather than a sentence — the
 * words belong to whoever renders the plex, through the `overflow` slot.
 */
const overflow = computed(() =>
  (Object.entries(frame.value.overflow) as [PlexRelatedSeat, number][]).filter(
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
  <div ref="frame" class="plex-frame numen" :data-moving="isMoving || undefined" :style="frameStyle">
    <PlexView
      ref="view"
      :frame="frame"
      :viewport="room"
      :node-size="options.nodeSize"
      :show-edge-labels="showEdgeLabels"
      :can-reach="canReach"
      :seat-name="seatName"
      :widen="widen"
      :hung="hung"
      :dwell="dwell"
      :reaching="reaching"
      :showing="showing"
      :clock="clock"
      :gesture-from="gesture.from.value"
      :gesture-at="gesture.at.value"
      :gesture-outcome="gesture.outcome.value"
      :dragged-at="dragging.at.value"
      :drop-seat="dragging.seat.value"
      :drop-name="dropName"
      @activate="emit('activate', $event)"
      @show="(id, showing) => emit('show', id, showing)"
      @reach="gesture.begin"
      @ask="gesture.ask"
      @menu="(id, at, opening) => emit('menu', id, at, opening)"
      @enter="(id, part) => emit('enter', id, part)"
    >
      <template v-if="$slots.icon" #icon="{ node }"><slot name="icon" :node="node" /></template>
    </PlexView>
    <div v-if="overflow.length" class="plex-frame__overflow" role="status">
      <slot name="overflow" :overflow="overflow">
        {{ overflow.map(([seat, count]) => countOf(seat, count)).join(', ') }} not shown
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
