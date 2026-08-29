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
import { Comment, computed, Fragment, ref, Text, useSlots, watch, type VNode } from 'vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import { isMenuKey, isPress, isShowKey } from './keys'
import { DWELL, useDwell, type Widened } from '../dwell'
import { byHandle, type Reaching } from '../reaching'
import { openedTo, woundBy, type HungParts, type Mark } from '../inside'
import { lerp } from '../arrange'
import { browserEnvironment, type Environment } from '../transition'
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
    /**
     * The box it widens to while the attention rests on it, and nothing where
     * it has no more of its title to show. How wide the whole title runs, and
     * how much window there is to grow into, are the picture's to work out.
     */
    wide?: Widened | null
    /**
     * The parts it hangs under its box while the attention rests, and nothing
     * for a node with none. What they are and what choosing one does are the
     * caller's.
     */
    hung?: HungParts | null
    /** How long the attention rests before it widens. Milliseconds. */
    dwell?: number
    /** How this node offers to be reached out of. The handle by default. */
    reaching?: Reaching
    /** The clock the opening is drawn on. Browser by default. */
    environment?: Environment
  }>(),
  {
    standing: 'open',
    wide: null,
    hung: null,
    dwell: DWELL,
    reaching: () => byHandle,
    environment: () => browserEnvironment,
  },
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
  /**
   * The attention has settled on this node, or has left it. A widened box is
   * drawn last of all, and which box that is only the whole picture knows.
   */
  (event: 'rest', resting: boolean): void
  /** A part of this node was chosen. The identifier is the caller's. */
  (event: 'enter', part: string): void
}>()

const over = ref(false)
const attended = ref(false)

const slots = useSlots()

/** Whether anything was drawn at all, which a placeholder and a blank are not. */
const anything = (drawn: readonly VNode[] | undefined): boolean =>
  !!drawn &&
  drawn.some((one) => {
    if (one.type === Comment) return false
    if (one.type === Fragment) return anything(one.children as VNode[])
    if (one.type === Text) return String(one.children).trim() !== ''
    return true
  })

/**
 * Whether this node is drawn something before its title. The caller answers
 * per node, and a node it draws nothing for keeps no room beside its title.
 */
const icon = computed(() => anything(slots.icon?.({ node: props.node })))

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

/**
 * What this node listens for beyond the handle, and whether it draws one.
 * Which of the two a reader gets is the plex's to choose.
 */
const reaching = computed(() => props.reaching)
const listening = reaching.value.listeners({
  ready: () => !ghost.value && props.standing === 'open',
  reach: (event: PointerEvent) => emit('reach', event),
})

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
 * Whether the keyboard is visibly on an element, which the browser works out
 * from how the focus got there. Where there is no such state, the answer is no.
 */
const keyboardOn = (element: Element) => {
  try {
    return element.matches(':focus-visible')
  } catch {
    return false
  }
}

/**
 * Where the attention is: the keyboard counts while it is the thing in use.
 * `focusout` carries where the focus went, so moving from the node onto its
 * own handle is not leaving.
 */
const attend = (event: FocusEvent) => {
  const within = event.currentTarget as Element
  if (event.type === 'focusout') {
    const next = event.relatedTarget as Node | null
    attended.value = !!(next && within.contains(next))
    return
  }
  attended.value = keyboardOn(event.target as Element)
}

/**
 * When there is a handle to press. Under the hand, or under the keyboard while
 * the keyboard is what is being used, or held there for as long as the gesture
 * that left from it lasts. A node on its way in or out offers nothing: it is
 * about to be somewhere else.
 */
const offering = computed(
  () =>
    props.reaching.handle &&
    props.node.opacity >= 1 &&
    (props.standing === 'source' ||
      (props.standing === 'open' && (over.value || attended.value))),
)

/** Whether there is anything to open: more of the title, or parts to hang. */
const opens = computed(() => !!props.wide || !!props.hung)

/**
 * What the attention is on, and where that stands. A box that moves under the
 * hand is somewhere else, and is settled on afresh.
 *
 * A gesture is under way at every standing but `open`, and nothing widens
 * while one is.
 */
const under = computed(() =>
  opens.value &&
  props.node.opacity >= 1 &&
  props.standing === 'open' &&
  (over.value || attended.value)
    ? `${props.node.x} ${props.node.y}`
    : null,
)

const open = useDwell(() => under.value, () => props.dwell, props.environment)

/** The box as it is drawn: the one it was placed with, opened towards the widened one. */
const box = computed<Widened>(() => {
  const wide = props.wide
  if (!wide || open.value <= 0) return { width: props.node.width, offset: 0 }
  return {
    width: lerp(props.node.width, wide.width, open.value),
    offset: lerp(0, wide.offset, open.value),
  }
})

/** Where the box begins, which everything drawn in it is placed from. */
const startsAt = computed(() => box.value.offset - box.value.width / 2)

/** How far the window on the parts has been wound down, counted in parts. */
const wound = ref(0)

/** The parts, as far out from under the box as they have come. */
const opened = computed(() =>
  props.hung ? openedTo(props.hung, open.value, wound.value) : null,
)

/** What a wheel moved that came to no whole part, held for the next one. */
let carried = 0

// The window opens at the top each time the attention settles afresh, and is
// left where it stands while the attention leaves.
watch(under, (now) => {
  if (now === null) return
  wound.value = 0
  carried = 0
})

/**
 * Winding the window over the parts. A wheel with nowhere to go is left to
 * whatever else wants it, and what it moved that came to no whole part is
 * carried into the next one.
 */
const wind = (event: WheelEvent) => {
  const hung = props.hung
  const shown = opened.value
  if (!hung || !shown) return

  const wheel = { delta: event.deltaY, mode: event.deltaMode }
  const { by, left } = woundBy(hung, wheel, carried)
  if (by === 0) {
    carried = left
    return
  }
  if (by < 0 ? !shown.above : !shown.below) {
    carried = 0
    return
  }

  carried = left
  event.preventDefault()
  event.stopPropagation()
  // Stepped from where the window really stands, which is the picture's own
  // reckoning of it.
  wound.value = shown.first + by
}

/** The line a mark at an edge is drawn along. */
const markLine = (mark: Mark) =>
  mark.points.map((at, index) => `${index === 0 ? 'M' : 'L'} ${at.x} ${at.y}`).join(' ')

/** A part chosen. */
const enter = (part: string) => emit('enter', part)

// A box that has begun to open is already over its neighbours.
watch(
  () => open.value > 0,
  (now) => emit('rest', now),
)

/** Where the handle sits. What it is made of is all sizes, and so all tokens. */
const handle = computed(() => {
  const at = handleIn({ ...props.node, width: box.value.width })
  return { x: at.x + box.value.offset, y: at.y }
})

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
    v-on="listening"
    @pointerenter="over = true"
    @pointerleave="over = false"
    @focusin="attend"
    @focusout="attend"
  >
    <rect
      class="plex__box"
      :x="startsAt"
      :y="-node.height / 2"
      :width="box.width"
      :height="node.height"
    />
    <foreignObject
      :x="startsAt"
      :y="-node.height / 2"
      :width="box.width"
      :height="node.height"
    >
      <div class="plex__title" :class="{ 'caps-numen': ghost }">
        <!-- Whatever stands for the thing a node addresses. The plex has no
             way to know what that is, so it is handed one. -->
        <span v-if="icon" class="plex__icon" aria-hidden="true">
          <slot name="icon" :node="node" />
        </span>
        <span class="plex__title-text">{{ node.title }}</span>
      </div>
    </foreignObject>

    <!-- The parts, come out from under the box. They are for the hand; the
         same parts are reached by name in the palette. -->
    <g v-if="hung && opened" class="plex__inside" aria-hidden="true" @wheel="wind">
      <!-- One ground under all of them, as deep as they have come. -->
      <rect
        class="plex__ground"
        :x="hung.offset - hung.width / 2"
        :y="hung.top"
        :width="hung.width"
        :height="opened.height"
        :opacity="opened.opacity"
      />
      <g
        v-for="part in opened.parts"
        :key="part.id"
        :opacity="part.opacity"
        :transform="`translate(0 ${hung.top + hung.pad + part.y})`"
      >
        <foreignObject
          :x="hung.offset - hung.width / 2 + hung.pad"
          y="0"
          :width="hung.width - 2 * hung.pad"
          :height="hung.partHeight"
        >
          <div
            class="plex__part"
            :style="{
              paddingInlineStart: `calc(var(--numen-node-padding) + ${part.indent}px)`,
            }"
            @click.stop="enter(part.id)"
            @dblclick.stop
          >
            <span class="plex__part-text">{{ part.text }}</span>
          </div>
        </foreignObject>
      </g>

      <!-- More of them than the window holds, the way they are wound to. -->
      <path
        v-for="mark in opened.marks"
        :key="mark.at"
        class="plex__more"
        :d="markLine(mark)"
        :opacity="opened.opacity"
      />
    </g>

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
  -webkit-user-select: none;
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

/* The ground the parts stand on: enough of it to hold them together, and thin
   enough to read the picture through. */
.plex__ground {
  rx: 0.25rem;
  fill: color-mix(in oklab, var(--numen-node-bg), transparent 25%);
  stroke: color-mix(in oklab, var(--numen-node-border), transparent 55%);
  stroke-width: var(--numen-stroke);
}

/* Each part is drawn in a box of its own, and where that box goes and how far
   it has faded up are SVG attributes on the group holding it. HTML inside a
   `foreignObject` that takes a layer of its own — under `opacity`, under
   `transform` — is drawn at the page's origin in WebKit, which is the engine
   the window is drawn in. */
.plex__part {
  block-size: 100%;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-edge-label-size);
  color: var(--numen-node-fg);
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  padding-inline-end: var(--numen-node-padding);
  border-radius: 0.1875rem;
  cursor: pointer;
  transition: background var(--numen-motion-hover) var(--numen-easing);
}

/* A ground under the one the hand is on, which is what says it can be pressed. */
.plex__part:hover {
  background: color-mix(in oklab, var(--numen-node-bg), var(--numen-node-fg) 12%);
}

/* One line, then an ellipsis, as a title is. */
.plex__part-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* There is more to wind to this way. */
.plex__more {
  fill: none;
  stroke: var(--numen-edge-label);
  stroke-width: var(--numen-stroke);
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
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

/* The focused node is painted from its own pair, and its seat's hue is the
   ground it stands on. What it is drawn before its title takes the ink the
   title is set in. */
.plex__node--focus .plex__icon {
  color: inherit;
}
</style>
