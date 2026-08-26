<script setup lang="ts">
/**
 * The plex: one node in focus, everything else placed by its seat.
 *
 * Takes a neighbourhood, gives back an identifier when one is chosen. It does
 * not fetch, does not know what an identifier addresses, and does not work out
 * who is related to whom. Answer `activate` with the next neighbourhood and it
 * travels there by itself.
 */
import {
  computed,
  onMounted,
  onScopeDispose,
  ref,
  toRef,
  useSlots,
  useTemplateRef,
  watch,
} from 'vue'
import PlexView from './render/PlexView.vue'
import { useTitleWidths } from './measure'
import { DWELL, widenedFor } from './dwell'
import { usePlexTransition, browserEnvironment, type Environment } from './transition'
import type { Placement, PlexOptionsInput } from './arrange'
import {
  countOf,
  seatWord,
  type PlacedNode,
  type PlexNeighbourhood,
  type PlexRelatedSeat,
  type PlexShowing,
  type Point,
} from './model'
import { resolveOptions } from './arrange'
import { usePlexGesture } from './gesture'
import type { MenuOpening } from '../menu/model'

const props = withDefaults(
  defineProps<{
    neighbourhood: PlexNeighbourhood
    options?: PlexOptionsInput
    /** Rows and columns unless another arrangement is handed in. */
    placement?: Placement
    showEdgeLabels?: boolean
    /** Milliseconds. Zero arrives instantly. */
    duration?: number
    /** The clock. Browser by default; a test hands in its own. */
    environment?: Environment
    /**
     * Seats a gesture may produce. A sibling is another of the parent's
     * children, so it is left out; which relationships exist is the caller's
     * to say.
     */
    creatable?: readonly PlexRelatedSeat[]
    /** How far a gesture travels before it is a drag and not a click. */
    dragThreshold?: number
    /**
     * How long the attention rests on a box before it widens to the whole of
     * its title. Milliseconds; nothing at all never widens.
     */
    dwell?: number
    /**
     * What to call a seat. The plex has to write one into the picture — the
     * outline a gesture draws says which seat it would take — and the words
     * for it belong to whoever renders the plex, as they do for the overflow
     * line. English by default, because something has to be drawn.
     */
    seatName?: (seat: PlexRelatedSeat) => string
  }>(),
  {
    showEdgeLabels: true,
    duration: 420,
    environment: () => browserEnvironment,
    creatable: () => ['parent', 'child', 'jump'],
    dragThreshold: 8,
    dwell: DWELL,
    seatName: seatWord,
  },
)

const emit = defineEmits<{
  /** A node other than the focus was chosen, by click or by keyboard. */
  (event: 'activate', id: string): void
  /**
   * A node asked for on its own: a double click, or a press with Shift held.
   * Where it is to be drawn is the second word, and the focus answers this as
   * every other node does.
   */
  (event: 'show', id: string, showing: PlexShowing): void
  /** Reached out into empty space: make a node in this seat of that one. */
  (event: 'create', from: string, seat: PlexRelatedSeat): void
  /** Reached out onto another node: relate the two in this seat. */
  (event: 'link', from: string, to: string, seat: PlexRelatedSeat): void
  /**
   * A menu was asked for on a node. The point is in the coordinates of the
   * screen; the element is what it was asked from, which is the only thing a
   * keypress hands over; the opening is what asked for it.
   *
   * Every node answers this, the focus included. What the menu holds and what
   * choosing an item does are the caller's.
   */
  (event: 'menu', id: string, at: Point, from: SVGGElement, opening: MenuOpening): void
  /** A menu asked for on a node has nothing left to stand on. */
  (event: 'dismiss'): void
}>()

/** What the window is taken to be until it has been measured. */
const FALLBACK = { width: 1200, height: 800 }

const frameElement = useTemplateRef<HTMLElement>('frame')
const viewport = ref(FALLBACK)

onMounted(() => {
  const element = frameElement.value
  if (!element || typeof ResizeObserver === 'undefined') return

  const observer = new ResizeObserver(([entry]) => {
    const box = entry?.contentRect
    if (box && box.width > 0 && box.height > 0) {
      viewport.value = { width: box.width, height: box.height }
    }
  })
  observer.observe(element)
  onScopeDispose(() => observer.disconnect())
})

/** Settled once and read by the measuring, the gesture and the drawing. */
const options = computed(() => resolveOptions({ ...props.options, viewport: viewport.value }))

const slots = useSlots()

/**
 * Room for the icon a caller draws beside a title, and none where the slot is
 * not filled.
 */
const iconRoom = computed(() => (slots.icon ? options.value.iconWidth : 0))

/**
 * How wide each title needs its box to be, and each label its line, measured
 * against the type the theme is written in. Taken before the first arrangement,
 * so a box is drawn at the size it keeps, and taken again when a theme changes
 * that type.
 */
const measures = useTitleWidths(() => iconRoom.value)

/**
 * How wide a box is drawn while the attention rests on it: the room its whole
 * title asks for, held inside the window.
 *
 * A title measured at no more than the box it is already in widens nothing,
 * and where there was nothing to measure the text with, nothing widens at all.
 */
const widen = computed(() => {
  const measure = measures.value?.node
  if (!measure) return undefined

  const { margin } = options.value
  const within = viewport.value
  return (node: PlacedNode) => widenedFor(node, measure(node), within, margin)
})

const { frame, moving } = usePlexTransition(
  () => props.neighbourhood,
  () => ({
    options: { ...props.options, viewport: viewport.value },
    placement: props.placement,
    measure: measures.value?.node,
    measureLabel: measures.value?.label,
    labelDepth: measures.value?.labelDepth,
  }),
  () => props.duration,
  props.environment,
)

/**
 * A caller that allows no seat at all has turned the gesture off, and a handle
 * that can come to nothing is a lie. Where a pointer happens to be is the
 * node's own affair, and never reaches this far.
 */
const mayReach = computed(() => props.creatable.length > 0)

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
const overflow = computed(
  () =>
    (Object.entries(frame.value.overflow) as [PlexRelatedSeat, number][]).filter(
      ([, count]) => count > 0,
    ),
)

defineExpose({ moving: toRef(moving) })
</script>

<template>
  <div
    ref="frame"
    class="plex-frame numen"
    :data-moving="moving || undefined"
    :style="{ '--numen-plex-move': `${duration}ms` }"
  >
    <PlexView
      :frame="frame"
      :viewport="viewport"
      :node-size="options.nodeSize"
      :show-edge-labels="showEdgeLabels"
      :may-reach="mayReach"
      :seat-name="seatName"
      :widen="widen"
      :dwell="dwell"
      :environment="environment"
      :gesture-from="gesture.from.value"
      :gesture-at="gesture.at.value"
      :gesture-outcome="gesture.outcome.value"
      @activate="emit('activate', $event)"
      @show="(id, showing) => emit('show', id, showing)"
      @reach="gesture.begin"
      @ask="gesture.ask"
      @menu="(id, at, from, opening) => emit('menu', id, at, from, opening)"
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
