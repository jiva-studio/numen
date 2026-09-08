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
import { byHandle, type ReachStrategy } from './reaching'
import { byDoubleClick, type ShowStrategy } from './showing'
import { hangParts, type PlexPart } from './inside'
import { usePlexTransition, browserClock, type Clock } from './transition'
import { browserViewport, type Viewport } from '@/shared/lib/viewport'
import type { Placement, PlexOptionsInput, Size } from './arrange'
import type { PlexNeighbourhood } from './neighbourhood'
import type { PlacedNode, Position } from './node'
import { countOf, seatWord, type PlexRelatedSeat } from './seat'
import type { PlexShowing } from './showing'
import { resolveOptions } from './arrange'
import { usePlexDrag } from './drag'
import { usePlexGesture } from './gesture'
import type { MenuOpening } from '@/shared/ui/menu'

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
    clock?: Clock
    /**
     * How much room the plex has, and what it becomes. Browser by default; a
     * test hands in its own and every coordinate is then a value it can name.
     */
    viewport?: Viewport
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
    /** How a node offers to be reached out of. The handle by default. */
    reaching?: ReachStrategy
    /** How a node is asked for on its own. The second click by default. */
    showing?: ShowStrategy
    /**
     * The parts of a node, asked for by the node's own identifier. They come
     * out from under its box while the attention rests on it, and a node named
     * none for hangs nothing.
     */
    parts?: (id: string) => readonly PlexPart[]
    /**
     * What to call a seat, for the outline a gesture draws and for the
     * overflow line. English by default.
     */
    seatName?: (seat: PlexRelatedSeat) => string
    /**
     * What is being dragged over the picture from somewhere else. Each
     * identifier is opaque and all of them are handed back untouched; an empty
     * list is nothing dragged, and the picture then draws none of it.
     */
    dragged?: readonly string[]
    /**
     * What to call what letting go with something dragged in would do. English
     * by default.
     */
    dropName?: (seat: PlexRelatedSeat) => string
  }>(),
  {
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
   * What was dragged in from outside was let go over the picture: relate each
   * of them to the focus in this seat. The identifiers are the ones they were
   * handed in as.
   */
  (event: 'bring', dragged: readonly string[], seat: PlexRelatedSeat): void
  /**
   * A menu was asked for on a node: which node, where on the screen, and what
   * asked for it. A keypress carries no point, so the middle of the box is
   * where it is asked.
   *
   * Every node answers this, the focus included. What the menu holds and what
   * choosing an item does are the caller's.
   */
  (event: 'menu', id: string, at: Position, opening: MenuOpening): void
  /** A menu asked for on a node has nothing left to stand on. */
  (event: 'dismiss'): void
  /**
   * A part of a node was chosen. Both identifiers are the caller's, handed
   * back as given.
   */
  (event: 'enter', id: string, part: string): void
}>()

defineSlots<{
  /** What is drawn beside a node's title. A node with none is drawn narrower. */
  icon?(props: { node: PlacedNode }): unknown
  /** What is said about the neighbours that did not fit, in the caller's words. */
  overflow?(props: { overflow: readonly [PlexRelatedSeat, number][] }): unknown
}>()

/** What the room is taken to be until it has been measured. */
const FALLBACK = { width: 1200, height: 800 }

const frameElement = useTemplateRef<HTMLElement>('frame')
/** The drawing, which a drag crossing the plex is measured against. */
const view = useTemplateRef<InstanceType<typeof PlexView>>('view')
/** How much room the plex has, as it was last measured. */
const room = ref<Size>(FALLBACK)

onMounted(() => {
  const element = frameElement.value
  if (!element) return

  onScopeDispose(
    props.viewport.watch(element, (size) => {
      room.value = size
    }),
  )
})

/** Settled once and read by the measuring, the gesture and the drawing. */
const options = computed(() => resolveOptions({ ...props.options, viewport: room.value }))

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
  const within = room.value
  return (node: PlacedNode) => widenedFor(node, measure(node), within, margin)
})

/**
 * The parts each node hangs under its box. The sizes are the ones the whole
 * picture is drawn to, so a plex set larger hangs them larger.
 */
const hung = computed(() => {
  const held = props.parts
  if (!held) return undefined

  const { margin } = options.value
  const deps = { measure: measures.value?.part, viewport: room.value, margin }
  return (node: PlacedNode) => hangParts(node, held(node.id), options.value, deps)
})

const { frame, moving } = usePlexTransition(
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
 * Something dragged across the picture from outside it.
 *
 * The plex works out which seat letting go comes to, measured from the focus,
 * and says so. What is being dragged it never looks at.
 */
const dragging = usePlexDrag({
  surface: () => view.value?.svg ?? null,
  dragged: () => props.dragged,
  frame: () => frame.value,
  options: () => options.value,
  viewport: () => room.value,
  allowed: () => props.creatable,
  threshold: () => props.dragThreshold,
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
const overflow = computed(
  () =>
    (Object.entries(frame.value.overflow) as [PlexRelatedSeat, number][]).filter(
      ([, count]) => count > 0,
    ),
)

defineExpose({
  moving: toRef(moving),
  /** The keyboard put back on a node by whoever took it away. */
  focusNode: (id: string) => view.value?.focusNode(id),
})
</script>

<template>
  <div
    ref="frame"
    class="plex-frame numen"
    :data-moving="moving || undefined"
    :style="{ '--numen-plex-move': `${duration}ms` }"
  >
    <PlexView
      ref="view"
      :frame="frame"
      :viewport="room"
      :node-size="options.nodeSize"
      :show-edge-labels="showEdgeLabels"
      :may-reach="mayReach"
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
