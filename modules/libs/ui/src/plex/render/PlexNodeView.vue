<script setup lang="ts">
/**
 * One node: its box, its title, and the handle to reach out from.
 *
 * Every number it draws with is already on the node it was handed. Where the
 * hand and the keyboard are is its own affair — the handle appears under
 * either — and the one thing it cannot work out is what it is to a gesture,
 * which arrives as its standing.
 *
 * The title goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import { computed, ref } from 'vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import { isMenuKey, isPress, isShowKey } from './keys'
import type { MenuOpening } from '../../menu/model'
import {
  handleIn,
  isReachable,
  isStop,
  nameOf,
  showingOf,
  type NodeStanding,
  type PlacedNode,
  type PlexShowing,
  type Point,
} from '../model'

const props = withDefaults(
  defineProps<{
    node: PlacedNode
    /** What this node is to the gesture. The one thing it cannot work out. */
    standing?: NodeStanding
  }>(),
  { standing: 'open' },
)

const emit = defineEmits<{
  /** Chosen, by click or by keyboard. Which node it was is the caller's to say. */
  (event: 'activate'): void
  /**
   * Asked to be drawn out on its own, and where it is to go. The modifier is
   * read here, so what travels on is the meaning.
   */
  (event: 'show', showing: PlexShowing): void
  /** A gesture began at the handle, and a pointer is dragging it somewhere. */
  (event: 'reach', pointer: PointerEvent): void
  /** The handle was pressed from the keyboard, where there is nowhere to drag. */
  (event: 'ask'): void
  /**
   * A menu was asked for on this node: where it was asked, the element it was
   * asked from, and what asked for it. A keypress carries no point, so it
   * carries both.
   */
  (event: 'menu', at: Point, from: SVGGElement, opening: MenuOpening): void
}>()

const over = ref(false)
const attended = ref(false)

/** Not a node yet, so nothing may be done to it and nothing is told about it. */
const ghost = computed(() => props.standing === 'ghost')

/** One predicate: the same rule decides the click and the name. */
const reachable = computed(() => !ghost.value && isReachable(props.node))

/** Where the keyboard stops: the focus too, and nothing on its way in or out. */
const stop = computed(() => !ghost.value && isStop(props.node))

/** The focus is announced although it cannot be chosen: it is where you are. */
const announced = computed(
  () => reachable.value || (!ghost.value && props.node.seat === 'focus'),
)

const activate = () => {
  if (reachable.value) emit('activate')
}

/**
 * Asking for the node itself, which every node that is really there answers —
 * the focus included, as it answers a menu.
 */
const show = (modified: boolean) => {
  if (stop.value) emit('show', showingOf(modified))
}

/** The middle of the node, for a press, which carries no point of its own. */
const middleOf = (element: SVGGElement): Point => {
  const box = element.getBoundingClientRect()
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 }
}

/** The webview draws a menu of its own over whatever does not refuse it. */
const onContextMenu = (event: MouseEvent) => {
  if (ghost.value) return
  event.preventDefault()
  emit(
    'menu',
    { x: event.clientX, y: event.clientY },
    event.currentTarget as SVGGElement,
    'pointer',
  )
}

const onKey = (event: KeyboardEvent) => {
  const group = event.currentTarget as SVGGElement
  if (isMenuKey(event)) {
    if (ghost.value) return
    event.preventDefault()
    emit('menu', middleOf(group), group, 'keyboard')
    return
  }
  if (isShowKey(event)) {
    event.preventDefault()
    show(event.altKey)
    return
  }
  if (!isPress(event)) return
  event.preventDefault()
  activate()
}

/**
 * Where the attention is, whichever way it arrived. `focusout` carries where
 * it went, so moving from the node onto its own handle is not leaving.
 */
const attend = (event: FocusEvent) => {
  const within = event.currentTarget as Element
  const next = event.relatedTarget as Node | null
  attended.value = event.type === 'focusin' || !!(next && within.contains(next))
}

/**
 * When there is a handle to press. Under the hand or under the keyboard, or
 * held there for as long as the gesture that left from it lasts. A node on
 * its way in or out offers nothing: it is about to be somewhere else.
 */
const offering = computed(
  () =>
    props.node.opacity >= 1 &&
    (props.standing === 'source' ||
      (props.standing === 'open' && (over.value || attended.value))),
)

/** Where the handle sits. What it is made of is all sizes, and so all tokens. */
const handle = computed(() => handleIn(props.node))

/**
 * One hue per seat, from a token named after it.
 *
 * There is no token for the focus, so on it this resolves to nothing and every
 * rule that reads the hue takes its fallback — which is how the focus comes to
 * wear its own colours rather than a seat's.
 */
const hue = computed(() => ({
  '--numen-seat-hue': `var(--numen-seat-${props.node.seat})`,
}))
</script>

<template>
  <g
    class="plex__node"
    :style="hue"
    :transform="`translate(${node.x} ${node.y})`"
    :opacity="node.opacity"
    :tabindex="stop ? 0 : -1"
    :aria-hidden="announced ? undefined : 'true'"
    :role="ghost ? undefined : node.seat === 'focus' ? 'img' : 'button'"
    :class="[`plex__node--${node.seat}`, `plex__node--${standing}`]"
    :aria-label="ghost ? undefined : nameOf(node)"
    @click="activate"
    @dblclick="show($event.altKey)"
    @contextmenu="onContextMenu"
    @keydown="onKey"
    @pointerenter="over = true"
    @pointerleave="over = false"
    @focusin="attend"
    @focusout="attend"
  >
    <rect
      class="plex__box"
      :x="-node.width / 2"
      :y="-node.height / 2"
      :width="node.width"
      :height="node.height"
    />
    <foreignObject
      :x="-node.width / 2"
      :y="-node.height / 2"
      :width="node.width"
      :height="node.height"
    >
      <div class="plex__title">
        <!-- Whatever stands for the thing a node addresses. The plex has no
             way to know what that is, so it is handed one. -->
        <span v-if="$slots.icon" class="plex__icon" aria-hidden="true">
          <slot name="icon" />
        </span>
        <span class="plex__title-text">{{ node.title }}</span>
      </div>
    </foreignObject>

    <!-- Reach out from here to make something. Under the hand or under the
         keyboard, so it is there when wanted and out of the way when not. -->
    <PlexNodeHandle
      v-if="offering"
      :at="handle"
      @reach="emit('reach', $event)"
      @ask="emit('ask')"
    />
  </g>
</template>

<style scoped>
/* No transition on the position — it comes from the frame, and a CSS one here
   would race it. */
.plex__node {
  cursor: pointer;
}

/* Hover mixes a little of a node's own text into the ground under it, which
   darkens a light node and lightens a dark one. The outline is left to the
   seat's hue, and the focused node is painted from the pair it wears. */
.plex__node:hover .plex__box {
  fill: color-mix(in oklab, var(--numen-node-bg), var(--numen-node-fg) 8%);
}

.plex__node--focus:hover .plex__box {
  fill: color-mix(in oklab, var(--numen-focus-bg), var(--numen-focus-fg) 8%);
}

.plex__node--focus {
  cursor: default;
}

/* The hue comes from the node's own seat, so a new seat needs a token and
   nothing here. The outline changes over the length of the move that changes
   the seat; the fill answers the pointer at the speed a pointer is answered. */
.plex__box {
  rx: var(--numen-plex-radius);
  fill: var(--numen-node-bg);
  stroke: var(--numen-seat-hue, var(--numen-node-border));
  stroke-width: var(--numen-stroke);
  transition:
    fill var(--numen-motion-hover) var(--numen-easing),
    stroke var(--numen-plex-move) var(--numen-easing);
}

/* While the plex is moving, the fill is a seat's colour too: the focused node
   is painted from its own pair, and follows the move as the outline does. */
[data-moving] .plex__box {
  transition:
    fill var(--numen-plex-move) var(--numen-easing),
    stroke var(--numen-plex-move) var(--numen-easing);
}

/* Icon then title, centred together in a box of a size the arrangement chose.

   Not selectable: a title is something to look at and press, and a drag that
   paints it blue is a drag that was meant to reach somewhere. */
.plex__title {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  padding-inline: var(--numen-node-padding);
  box-sizing: border-box;
  color: var(--numen-node-fg);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  user-select: none;
  transition: color var(--numen-plex-move) var(--numen-easing);
}

.plex__icon {
  flex: none;
  display: flex;
  align-items: center;
  color: var(--numen-seat-hue, var(--numen-node-fg));
}

/* One line, then an ellipsis. Two lines cost as much height again for a title
   that is a sentence, and a box that grows is a box the arrangement did not
   plan for. */
.plex__title-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* The node a link would be made to, while the pointer is still on it. */
.plex__node--target .plex__box {
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

/* Not there yet: an outline where a node would appear, and out of the way of
   everything under it — including the gesture still looking for somewhere to
   land. What it says is the seat, not a name, because it has none. */
.plex__node--ghost {
  cursor: default;
  pointer-events: none;
}

.plex__node--ghost .plex__box {
  fill: none;
  stroke: var(--numen-seat-hue, var(--numen-ring));
  stroke-dasharray: var(--numen-ghost-dash);
  transition: none;
}

.plex__node--ghost .plex__title {
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  text-transform: uppercase;
  letter-spacing: var(--numen-caps-tracking);
}

.plex__node:focus-visible {
  outline: none;
}

.plex__node:focus-visible .plex__box {
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

.plex__node--focus .plex__box {
  rx: var(--numen-plex-radius-focus);
  fill: var(--numen-focus-bg);
  stroke: var(--numen-focus-border);
}

.plex__node--focus .plex__title {
  color: var(--numen-focus-fg);
}
</style>
