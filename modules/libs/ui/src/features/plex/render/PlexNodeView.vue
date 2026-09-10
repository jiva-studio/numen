<script setup lang="ts">
/**
 * One node: its box, its title, and the handle to reach out from. Every number
 * it draws with is already on the node it was handed, but for what it is to a
 * gesture, which arrives as its role in one.
 *
 * The title goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import {
  Comment,
  computed,
  Fragment,
  ref,
  Text,
  useSlots,
  useTemplateRef,
  watch,
  type VNode,
} from 'vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import PlexNodeParts from './PlexNodeParts.vue'
import { isMenuKey, isPress, isShowKey } from './keys'
import { boxOf, DWELL, useDwell, type WideBox } from '../dwell'
import { byHandle, type ReachStrategy } from '../reaching'
import { byDoubleClick, joined, showingOf, type PlexShowing, type ShowStrategy } from '../showing'
import type { HungParts } from '../inside'
import { browserClock, type Clock } from '../transition'
import type { MenuOpening } from '@/shared/ui/menu'
import {
  handleIn,
  isReachable,
  isStop,
  nameOf,
  type GestureRole,
  type PlacedNode,
  type Position,
} from '../node'

// --- Props & Emits ---
const props = withDefaults(
  defineProps<{
    node: PlacedNode
    /** What this node is to the gesture. The one thing it cannot work out. */
    gestureRole?: GestureRole
    /**
     * The box it widens to while the attention rests on it, and nothing where
     * it has no more of its title to show.
     */
    wide?: WideBox | null
    /**
     * The parts it hangs under its box while the attention rests, and nothing
     * for a node with none.
     */
    hung?: HungParts | null
    /** How long the attention rests before it widens. Milliseconds. */
    dwell?: number
    /** How this node offers to be reached out of. The handle by default. */
    reaching?: ReachStrategy
    /** How this node is asked for on its own. The second click by default. */
    showing?: ShowStrategy
    /** The clock the opening is drawn on. Browser by default. */
    clock?: Clock
  }>(),
  {
    gestureRole: 'open',
    wide: null,
    hung: null,
    dwell: DWELL,
    reaching: () => byHandle,
    showing: () => byDoubleClick,
    clock: () => browserClock,
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
   * A menu was asked for on this node: where it was asked, and what asked for
   * it. A keypress carries no point of its own, so the middle of the box is
   * where it is asked.
   */
  (event: 'menu', at: Position, opening: MenuOpening): void
  /**
   * The attention has settled on this node, or has left it. A widened box is
   * drawn last of all, and which box that is only the whole picture knows.
   */
  (event: 'rest', resting: boolean): void
  /** A part of this node was chosen. The identifier is the caller's. */
  (event: 'enter', part: string): void
}>()

defineSlots<{
  /** What is drawn beside this node's title. */
  icon?(props: { node: PlacedNode }): unknown
}>()

const group = useTemplateRef<SVGGElement>('group')

/** The keyboard put back on this node by whoever took it away. */
defineExpose({ focus: () => group.value?.focus() })

// --- State ---
const isOver = ref(false)
const isAttended = ref(false)

const slots = useSlots()

/**
 * Whether this node is drawn something before its title.
 */
const hasIcon = computed(() => hasAnything(slots.icon?.({ node: props.node })))

/** Not a node yet, so nothing may be done to it and nothing is told about it. */
const isGhost = computed(() => props.gestureRole === 'ghost')

/** One predicate: the same rule decides the click and the name. */
const canReach = computed(() => !isGhost.value && isReachable(props.node))

/** Where the keyboard stops: the focus too, and nothing on its way in or out. */
const canStop = computed(() => !isGhost.value && isStop(props.node))

/** The focus is announced although it cannot be chosen: it is where you are. */
const isAnnounced = computed(
  () => canReach.value || (!isGhost.value && props.node.seat === 'focus'),
)

/**
 * When there is a handle to press. Under the hand, or under the keyboard while
 * the keyboard is what is being used.
 */
const isOffering = computed(
  () =>
    props.reaching.handle &&
    props.node.opacity >= 1 &&
    (props.gestureRole === 'source' ||
      (props.gestureRole === 'open' && (isOver.value || isAttended.value))),
)

/** Whether there is anything to open: more of the title, or parts to hang. */
const canOpen = computed(() => !!props.wide || !!props.hung)

/**
 * What the attention is on, and where that stands.
 */
const under = computed(() =>
  canOpen.value &&
  props.node.opacity >= 1 &&
  props.gestureRole === 'open' &&
  (isOver.value || isAttended.value)
    ? `${props.node.x} ${props.node.y}`
    : null,
)

const open = useDwell(() => under.value, () => props.dwell, props.clock)

const box = computed(() => boxOf(props.node, props.wide, open.value))

/** Where the box begins, which everything drawn in it is placed from. */
const startsAt = computed(() => box.value.offset - box.value.width / 2)

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
 */
const hue = computed(() => ({
  '--numen-seat-hue': `var(--numen-seat-${props.node.seat})`,
}))

/**
 * What this node listens for beyond the handle, and whether it draws one.
 */
const listening = joined(
  props.reaching.listeners({
    ready: () => !isGhost.value && props.gestureRole === 'open',
    reach: (event: PointerEvent) => emit('reach', event),
  }),
  props.showing.listeners({
    ready: () => canStop.value,
    show: (modified: boolean) => showNode(modified),
  }),
)

// --- Handlers ---
function onClick(): void {
  activateNode()
}

function onDoubleClick(event: MouseEvent): void {
  if (props.showing.doubleClick) {
    showNode(event.altKey)
  }
}

function onContextMenu(event: MouseEvent): void {
  if (isGhost.value) return
  event.preventDefault()
  emit('menu', { x: event.clientX, y: event.clientY }, 'pointer')
}

function onKeyDown(event: KeyboardEvent): void {
  if (isMenuKey(event)) {
    if (isGhost.value) return
    event.preventDefault()
    const element = group.value
    if (element) emit('menu', getMiddleOf(element), 'keyboard')
    return
  }
  if (isShowKey(event)) {
    event.preventDefault()
    showNode(event.altKey)
    return
  }
  if (!isPress(event)) return
  event.preventDefault()
  activateNode()
}

function onPointerEnter(): void {
  isOver.value = true
}

function onPointerLeave(): void {
  isOver.value = false
}

function onFocusIn(event: FocusEvent): void {
  updateAttendance(event)
}

function onFocusOut(event: FocusEvent): void {
  updateAttendance(event)
}

function onPartEnter(part: string): void {
  emit('enter', part)
}

function onHandleReach(event: PointerEvent): void {
  emit('reach', event)
}

function onHandleAsk(): void {
  emit('ask')
}

// --- Helpers ---
function activateNode(): void {
  if (canReach.value) emit('activate')
}

function showNode(modified: boolean): void {
  if (canStop.value) emit('show', showingOf(modified))
}

/** Whether anything was drawn at all, which a placeholder and a blank are not. */
function hasAnything(drawn: readonly VNode[] | undefined): boolean {
  return (
    !!drawn &&
    drawn.some((one) => {
      if (one.type === Comment) return false
      if (one.type === Fragment) return hasAnything(one.children as VNode[])
      if (one.type === Text) return String(one.children).trim() !== ''
      return true
    })
  )
}

/** The middle of the node, for a press, which carries no point of its own. */
function getMiddleOf(element: SVGGElement): Position {
  const box = element.getBoundingClientRect()
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 }
}

function isKeyboardOn(element: Element): boolean {
  try {
    return element.matches(':focus-visible')
  } catch {
    return false
  }
}

function updateAttendance(event: FocusEvent): void {
  const within = event.currentTarget as Element
  if (event.type === 'focusout') {
    const next = event.relatedTarget as Node | null
    isAttended.value = !!(next && within.contains(next))
    return
  }
  isAttended.value = isKeyboardOn(event.target as Element)
}
</script>

<template>
  <g
    ref="group"
    class="plex__node"
    :style="hue"
    :transform="`translate(${node.x} ${node.y})`"
    :opacity="node.opacity"
    :tabindex="canStop ? 0 : -1"
    :aria-hidden="isAnnounced ? undefined : 'true'"
    :role="isGhost ? undefined : node.seat === 'focus' ? 'img' : 'button'"
    :class="[`plex__node--${node.seat}`, `plex__node--${gestureRole}`]"
    :aria-label="isGhost ? undefined : nameOf(node)"
    @click="onClick"
    @dblclick="onDoubleClick"
    @contextmenu="onContextMenu"
    @keydown="onKeyDown"
    v-on="listening"
    @pointerenter="onPointerEnter"
    @pointerleave="onPointerLeave"
    @focusin="onFocusIn"
    @focusout="onFocusOut"
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
      <div class="plex__title" :class="{ 'caps-numen': isGhost }">
        <!-- Whatever stands for the thing a node addresses. The plex has no
             way to know what that is, so it is handed one. -->
        <span v-if="hasIcon" class="plex__icon" aria-hidden="true">
          <slot name="icon" :node="node" />
        </span>
        <span class="plex__title-text">{{ node.title }}</span>
      </div>
    </foreignObject>

    <PlexNodeParts
      v-if="hung"
      :hung="hung"
      :open="open"
      @enter="onPartEnter"
    />

    <!-- Reach out from here to make something. Under the hand or under the
         keyboard, so it is there when wanted and out of the way when not. -->
    <PlexNodeHandle
      v-if="isOffering"
      :at="handle"
      @reach="onHandleReach"
      @ask="onHandleAsk"
    />
  </g>
</template>

<style scoped>
/* The position comes from the frame, and carries no transition of its own. */
.plex__node {
  /* The corner of a node's box, and the corner of the box around the one in
     front. */
  --radius: 0.375rem;
  --radius-focus: 0.5rem;
  /* The dash the outline of a node that is not there yet is drawn in. */
  --ghost-dash: 6 4;

  cursor: pointer;
}

/* Hover mixes a little of a node's own text into the ground under it, which
   darkens a light node and lightens a dark one. The outline is left to the
   seat's hue, and the focused node is painted from the pair it wears. */
.plex__node:hover .plex__box {
  fill: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 8%);
}

.plex__node--focus:hover .plex__box {
  fill: color-mix(in oklab, var(--numen-accent), var(--numen-accent-ink) 8%);
}

.plex__node--focus {
  cursor: default;
}

/* The hue comes from the node's own seat, so a new seat needs a token and
   nothing here. The outline changes over the length of the move that changes
   the seat; the fill answers the pointer at the speed a pointer is answered. */
.plex__box {
  rx: var(--radius);
  fill: var(--numen-raised);
  stroke: var(--numen-seat-hue, var(--numen-rule));
  stroke-width: var(--numen-stroke);
  transition:
    fill var(--numen-motion-hover) var(--numen-easing),
    stroke var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

/* While the plex is moving, the fill is a seat's colour too: the focused node
   is painted from its own pair, and follows the move as the outline does. */
[data-moving] .plex__box {
  transition:
    fill var(--numen-plex-move, var(--numen-motion)) var(--numen-easing),
    stroke var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

/* Icon then title, centred together in a box of a size the arrangement chose. A
   title is something to look at and press, and takes no selection. */
.plex__title {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  padding-inline: var(--numen-node-padding);
  box-sizing: border-box;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  user-select: none;
  -webkit-user-select: none;
  transition: color var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

.plex__icon {
  flex: none;
  display: flex;
  align-items: center;
  color: var(--numen-seat-hue, var(--numen-ink));
}

/* One line, then an ellipsis. A box stands at the height the arrangement gave
   it, whatever its title runs to. */
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
  stroke-dasharray: var(--ghost-dash);
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
  outline: none;
}

.plex__node--focus .plex__box {
  rx: var(--radius-focus);
  fill: var(--numen-accent);
  stroke: var(--numen-accent);
}

.plex__node--focus .plex__title {
  color: var(--numen-accent-ink);
}

/* The focused node is painted from its own pair, and its seat's hue is the
   ground it stands on. What it is drawn before its title takes the ink the
   title is set in. */
.plex__node--focus .plex__icon {
  color: inherit;
}
</style>
