<script setup lang="ts">
/**
 * What a thing is, said beside it while a person points at it.
 *
 * It follows the pointer and never takes it: a tooltip a mouse can land on is
 * one that moves out from under the hand reaching for what it is about.
 *
 * It stands on the far side of the thing it is about, takes the near side
 * where the far one has no room for it, and is brought inside the edge where
 * neither side has. A menu is placed the same way.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

import { beside, type Box } from '../placing/place'
import type { Size } from '../plex/arrange'

const props = withDefaults(
  defineProps<{
    /** The thing it is about, in pixels from the top left of the window. */
    at: Box
    /** The area it is placed in. The browser's own by default. */
    viewport?: Size | null
    /** Kept clear of that area's edges. */
    margin?: number
    /** Left between it and the thing it is about. */
    gap?: number
  }>(),
  { viewport: null, margin: 8, gap: 8 },
)

const held = ref<HTMLElement | null>(null)

/** Its own size, which only the drawing knows. Placement is worked out from it. */
const size = ref<Size>({ width: 0, height: 0 })

/** The window, measured, and measured again whenever it changes size. */
const window_ = ref<Size>({ width: window.innerWidth, height: window.innerHeight })

/** The area to stay inside. The browser's, unless a caller measures its own. */
const room = computed<Size>(() => props.viewport ?? window_.value)

/**
 * Across, it stands beside the thing it is about. Down, it begins where that
 * thing begins, and folds up from there at the foot of the window.
 */
const placed = computed(() => ({
  x: beside({
    from: props.at.x,
    to: props.at.x + props.at.width,
    size: size.value.width,
    room: room.value.width,
    margin: props.margin,
    gap: props.gap,
  }),
  y: beside({
    from: props.at.y,
    to: props.at.y,
    size: size.value.height,
    room: room.value.height,
    margin: props.margin,
    gap: 0,
  }),
}))

const measure = () => {
  const box = held.value?.getBoundingClientRect()
  if (box) size.value = { width: box.width, height: box.height }
}

const resized = () => {
  window_.value = { width: window.innerWidth, height: window.innerHeight }
  measure()
}

onMounted(() => {
  measure()
  window.addEventListener('resize', resized)
})

onBeforeUnmount(() => window.removeEventListener('resize', resized))

// Measured again once the drawing has caught up with the thing it is about.
watch(() => props.at, measure, { flush: 'post' })
</script>

<template>
  <aside
    ref="held"
    class="tooltip"
    role="tooltip"
    :style="{ insetInlineStart: `${placed.x}px`, insetBlockStart: `${placed.y}px` }"
  >
    <slot />
  </aside>
</template>

<style scoped>
.tooltip {
  position: fixed;
  z-index: 1;
  /* As wide as what it says, whatever room is left beside where it stands. */
  inline-size: max-content;
  padding: var(--numen-inset);
  border: 1px solid var(--numen-node-border);
  border-radius: var(--numen-radius);
  background: var(--numen-node-bg);
  color: var(--numen-node-fg);
  box-shadow: var(--numen-shadow-raised, 0 2px 8px rgb(0 0 0 / 25%));
  font-size: var(--numen-text-1);
  pointer-events: none;
}
</style>
