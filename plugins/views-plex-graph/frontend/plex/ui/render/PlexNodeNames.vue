<script setup lang="ts">
/**
 * The names set over and under one node, in small faint text with no pill. They sit
 * back, at the resting opacity, until the pointer is over the node or over one of them, which
 * brings them to the focused opacity. What
 * choosing one does is the caller's.
 */
import { computed } from 'vue'
import type { Attention } from './props'
import {
  DEFAULT_FOCUSED_NAMES_OPACITY,
  DEFAULT_NAMES_OPACITY,
  placeNames,
  ROW_RATIO,
  type NamesRoom,
} from '../../lib/names'
import type { PlacedNode } from '../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    node: PlacedNode
    room: NamesRoom
    attention: Attention
    restingOpacity?: number
    focusedOpacity?: number
  }>(),
  { restingOpacity: DEFAULT_NAMES_OPACITY, focusedOpacity: DEFAULT_FOCUSED_NAMES_OPACITY },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A name was chosen. The identifier is the caller's, returned unchanged. */
  (event: 'choose', id: string): void
  /** The pointer entered or left a name. */
  (event: 'attend', isAttended: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
/** Whether the pointer is over the node or over one of its names. */
const isForward = computed(
  () => props.attention.node === props.node.id || props.attention.names === props.node.id,
)

const placed = computed(() => placeNames(props.room, props.node))

const opacity = computed(() => (isForward.value ? props.focusedOpacity : props.restingOpacity))

/** The type of a name is the size its row was measured at. */
const fontSize = computed(() => `${props.room.rowHeight / ROW_RATIO}px`)
</script>

<template>
  <g class="plex__names" :opacity="node.opacity" :style="{ fontSize }">
    <g
      v-for="name in placed"
      :key="name.id"
      class="plex__name"
      :opacity="opacity"
      role="button"
      tabindex="0"
      :aria-label="name.title"
      :transform="`translate(${name.x} ${name.y})`"
      @click="emit('choose', name.id)"
      @pointerenter="emit('attend', true)"
      @pointerleave="emit('attend', false)"
      @keydown.enter.prevent="emit('choose', name.id)"
      @keydown.space.prevent="emit('choose', name.id)"
    >
      <rect
        class="plex__name-hit"
        :x="-name.width / 2"
        :y="-name.height / 2"
        :width="name.width"
        :height="name.height"
      />
      <text class="plex__name-text" text-anchor="middle" dominant-baseline="central">
        {{ name.title }}
      </text>
    </g>
  </g>
</template>

<style scoped>
.plex__names {
  pointer-events: none;
}

.plex__name {
  cursor: pointer;
  pointer-events: all;
  transition: opacity var(--plex-move) var(--numen-easing);
}

.plex__name-hit {
  fill: transparent;
}

.plex__name-text {
  fill: var(--numen-hushed);
  font-family: var(--numen-font-sans);
}

.plex__name:hover .plex__name-text,
.plex__name:focus-visible .plex__name-text {
  fill: var(--numen-ink);
}

.plex__name:focus-visible {
  outline: none;
}
</style>
