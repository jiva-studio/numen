<script setup lang="ts">
/**
 * One node: its box, its title, and the handle to reach out from. Every number
 * it draws with is already on the node it was handed, but for what it is to a
 * gesture, which arrives as its role in one.
 */
import { computed, useTemplateRef, watch } from 'vue'
import PlexNodeBox from './PlexNodeBox.vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import PlexNodeHeadings from './PlexNodeHeadings.vue'
import type { PlexDrawnSlots, PlexNodeEvents, PlexNodeProps } from './props'
import { useHoverFocus } from '../../model/hoverFocus'
import { DEFAULT_BEHAVIOUR } from '../../model/behaviour'
import { useOpenBox } from '../../model/expansion'
import { useNodePress } from '../../model/press'
import { createAriaLabel, type Position } from '../../lib/node'

const props = withDefaults(defineProps<PlexNodeProps>(), {
  gestureRole: 'open',
  wide: null,
  hung: null,
  behaviour: () => DEFAULT_BEHAVIOUR,
})

const emit = defineEmits<PlexNodeEvents>()

defineSlots<PlexDrawnSlots>()

const group = useTemplateRef<SVGGElement>('group')

/** The keyboard put back on this node by whoever took it away. */
defineExpose({ focus: () => group.value?.focus() })

const hoverFocus = useHoverFocus()

/** The middle of the node, for a press, which carries no point of its own. */
const getMiddle = (): Position | null => {
  const element = group.value
  if (!element) return null
  const box = element.getBoundingClientRect()
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 }
}

const press = useNodePress(props, getMiddle, {
  activate: () => emit('activate'),
  show: (showing) => emit('show', showing),
  reach: (pointer) => emit('reach', pointer),
  menu: (at, trigger) => emit('menu', at, trigger),
})

const { isGhost, canStop, isAnnounced, listening } = press

/**
 * When there is a handle to press. Under the hand, or under the keyboard while
 * the keyboard is what is being used.
 */
const hasDropTarget = computed(
  () =>
    props.behaviour.reaching.hasHandle &&
    props.node.opacity >= 1 &&
    (props.gestureRole === 'source' || (props.gestureRole === 'open' && hoverFocus.isOn.value)),
)

/** A node that is not there yet is announced as nothing; the focus is a picture. */
const role = computed(() => {
  if (isGhost.value) return undefined
  return props.node.relation === 'focus' ? 'img' : 'button'
})

/** Whether there is anything to open: more of the title, or headings to hang. */
const canOpen = computed(() => !!props.wide || !!props.hung)

/**
 * What is hovered or focused, and where that stands.
 */
const openPosition = computed(() =>
  canOpen.value && props.node.opacity >= 1 && props.gestureRole === 'open' && hoverFocus.isOn.value
    ? `${props.node.x} ${props.node.y}`
    : null,
)

const { open, box, handle } = useOpenBox({
  node: () => props.node,
  wide: () => props.wide,
  on: () => openPosition.value,
  delay: () => props.behaviour.expansionDelay,
  clock: props.behaviour.clock,
})

// A box that has begun to open is already over its neighbours.
watch(
  () => open.value > 0,
  (now) => emit('settle', now),
)

/**
 * One hue per relation, from a token named after it.
 */
const hue = computed(() => ({
  '--numen-relation-hue': `var(--numen-relation-${props.node.relation})`,
}))
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
    :role="role"
    :class="[`plex__node--${node.relation}`, `plex__node--${gestureRole}`]"
    :aria-label="isGhost ? undefined : createAriaLabel(node)"
    @click="press.onClick"
    @dblclick="press.onDoubleClick"
    @contextmenu="press.onContextMenu"
    @keydown="press.onKeyDown"
    v-on="listening"
    @pointerenter="hoverFocus.onPointerEnter"
    @pointerleave="hoverFocus.onPointerLeave"
    @focusin="hoverFocus.onFocusIn"
    @focusout="hoverFocus.onFocusOut"
  >
    <PlexNodeBox :node="node" :box="box" :is-ghost="isGhost">
      <template v-if="$slots.icon" #icon="{ node: drawn }">
        <slot name="icon" :node="drawn" />
      </template>
    </PlexNodeBox>

    <PlexNodeHeadings
      v-if="hung"
      :hung="hung"
      :open="open"
      @enter="(heading) => emit('enter', heading)"
    />

    <!-- Reach out from here to make something. Under the hand or under the
         keyboard, so it is there when wanted and out of the way when not. -->
    <PlexNodeHandle
      v-if="hasDropTarget"
      :at="handle"
      @reach="(pointer) => emit('reach', pointer)"
      @ask="emit('ask')"
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
   relation's hue, and the focused node is painted from the pair it wears. */
.plex__node:hover :deep(.plex__box) {
  fill: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 8%);
}

.plex__node--focus:hover :deep(.plex__box) {
  fill: color-mix(in oklab, var(--numen-accent), var(--numen-accent-ink) 8%);
}

.plex__node--focus {
  cursor: default;
}

[data-moving] .plex__node :deep(.plex__box) {
  transition:
    fill var(--numen-plex-move, var(--numen-motion)) var(--numen-easing),
    stroke var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

/* The node a link would be made to, while the pointer is still on it. */
.plex__node--target :deep(.plex__box) {
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

/* Not there yet: an outline where a node would appear, and out of the way of
   everything under it — including the gesture still looking for somewhere to
   land. What it says is the relation, not a name, because it has none. */
.plex__node--ghost {
  cursor: default;
  pointer-events: none;
}

.plex__node--ghost :deep(.plex__box) {
  fill: none;
  stroke: var(--numen-relation-hue, var(--numen-ring));
  stroke-dasharray: var(--ghost-dash);
  transition: none;
}

.plex__node--ghost :deep(.plex__title) {
  color: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
}

.plex__node:focus-visible {
  outline: none;
}

.plex__node:focus-visible :deep(.plex__box) {
  outline: none;
}

.plex__node--focus :deep(.plex__box) {
  rx: var(--radius-focus);
  fill: var(--numen-accent);
  stroke: var(--numen-accent);
}

.plex__node--focus :deep(.plex__title) {
  color: var(--numen-accent-ink);
}

.plex__node--focus :deep(.plex__icon) {
  color: inherit;
}
</style>
