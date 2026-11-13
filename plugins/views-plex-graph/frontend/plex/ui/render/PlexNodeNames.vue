<script setup lang="ts">
/**
 * The names set over and under one node, in small faint text with no pill. They sit back at the
 * resting opacity until the pointer is over the node or over the room of its names, which brings
 * them forward: at the focused opacity, and then they can be chosen. What choosing one does is the caller's.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { Attention } from '../../model/useAttention'
import { placeNames, type NamesRoom } from '../../lib/names'
import { createNameLabel, type PlacedNode } from '../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  node: PlacedNode
  room: NamesRoom
  attention: Attention
  restingOpacity: number
  focusedOpacity: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A name was chosen. The identifier is the caller's, returned unchanged. */
  (event: 'choose', id: string): void
  /** The pointer entered or left the room of a name. */
  (event: 'attend', isAttended: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
/** Whether the pointer is over the node or over its names. */
const isForward = computed(
  () => props.attention.node === props.node.id || props.attention.names === props.node.id,
)

const placed = computed(() => placeNames(props.room, props.node))

/** The name the pointer is on, which is none once that name is gone. */
const overId = ref<string | null>(null)

const opacity = computed(() => (isForward.value ? props.focusedOpacity : props.restingOpacity))

/* --------------------------------- Hooks ---------------------------------- */
onBeforeUnmount(() => emit('attend', false))
watch(placed, (now) => {
  if (overId.value === null || now.some((name) => name.id === overId.value)) return
  overId.value = null
  emit('attend', false)
})

/* -------------------------------- Handlers -------------------------------- */
// A click with no count and no pointer over the name comes from assistive technology, which has no pointer to bring the names forward.
function onChoose(id: string, event: MouseEvent) {
  if (isForward.value || (event.detail === 0 && overId.value === null)) emit('choose', id)
}

function onEnter(id: string) {
  overId.value = id
  emit('attend', true)
}

function onLeave(id: string) {
  if (overId.value === id) overId.value = null
  emit('attend', false)
}
</script>

<template>
  <g class="plex__names" :opacity="node.opacity" :style="{ fontSize: `${room.fontSize}px` }">
    <g
      v-for="name in placed"
      :key="name.id"
      class="plex__name"
      :class="{ 'plex__name--forward': isForward }"
      :opacity="opacity"
      role="button"
      :aria-label="createNameLabel(name.title, node)"
      :transform="`translate(${name.x} ${name.y})`"
      @click="onChoose(name.id, $event)"
      @pointerenter="onEnter(name.id)"
      @pointerleave="onLeave(name.id)"
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
  pointer-events: all;
  transition: opacity var(--plex-move) var(--numen-easing);
}

.plex__name--forward {
  cursor: pointer;
}

.plex__name-hit {
  fill: transparent;
}

.plex__name-text {
  fill: var(--numen-hushed);
  font-family: var(--numen-font-sans);
}

.plex__name--forward:hover .plex__name-text {
  fill: var(--numen-ink);
}
</style>
