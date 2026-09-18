<script setup lang="ts">
/**
 * What a thing is, said beside it while a person points at it. It follows the
 * pointer and never takes it.
 *
 * It stands on the far side of the thing it is about, takes the near side where
 * the far one has no room, and is brought inside the edge where neither has.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'

import { getPlaceBeside, type Box } from '@/shared/lib/place'
import type { Size } from '@/shared/lib/geometry'
import { browserViewport, type Viewport } from '@/shared/lib/viewport'

const props = withDefaults(
  defineProps<{
    /** The thing it is about, in pixels from the top left of the window. */
    at: Box
    /** What measures the area it is placed in. The browser by default. */
    viewport?: Viewport
    /** Kept clear of that area's edges. */
    margin?: number
    /** Left between it and the thing it is about. */
    gap?: number
  }>(),
  { viewport: () => browserViewport, margin: 8, gap: 8 },
)

const root = useTemplateRef<HTMLElement>('root')

/** Its own size, which only the drawing knows. Placement is worked out from it. */
const size = ref<Size>({ width: 0, height: 0 })

/** The area to stay inside, measured again whenever it changes. */
const room = ref<Size>({ width: 0, height: 0 })

/**
 * Across, it stands beside the thing it is about. Down, it begins where that
 * thing begins, and folds up from there at the foot of the window.
 */
const placed = computed(() => ({
  x: getPlaceBeside({
    from: props.at.x,
    to: props.at.x + props.at.width,
    size: size.value.width,
    room: room.value.width,
    margin: props.margin,
    gap: props.gap,
  }),
  y: getPlaceBeside({
    from: props.at.y,
    to: props.at.y,
    size: size.value.height,
    room: room.value.height,
    margin: props.margin,
    gap: 0,
  }),
}))

const tooltipStyle = computed(() => ({
  insetInlineStart: `${placed.value.x}px`,
  insetBlockStart: `${placed.value.y}px`,
}))

const measure = () => {
  const box = root.value?.getBoundingClientRect()
  if (box) size.value = { width: box.width, height: box.height }
}

/** What stops the watching, held from the moment it begins. */
let stopWatching: (() => void) | null = null

onMounted(() => {
  measure()
  stopWatching = props.viewport.watchRoom((size) => {
    room.value = size
    measure()
  })
})

onBeforeUnmount(() => stopWatching?.())

// Measured again once the drawing has caught up with the thing it is about.
watch(() => props.at, measure, { flush: 'post' })
</script>

<template>
  <aside ref="root" class="tooltip" role="tooltip" :style="tooltipStyle">
    <slot />
  </aside>
</template>

<style scoped>
.tooltip {
  position: fixed;
  z-index: var(--numen-lift-tooltip);
  /* As wide as what it says, whatever room is left beside where it stands. */
  inline-size: max-content;
  padding: var(--numen-inset);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background: var(--numen-raised);
  color: var(--numen-ink);
  box-shadow: var(--numen-shadow-card);
  font-size: var(--numen-text-1);
  pointer-events: none;
}
</style>
