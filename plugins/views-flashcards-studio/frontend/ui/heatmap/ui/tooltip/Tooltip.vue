<script setup lang="ts">
import { measureRect } from '../../../../dom'
/**
 * A label for the hovered item, displayed beside it. It follows the pointer and
 * never captures it.
 *
 * It is placed on the far side of the item it describes, uses the near side
 * where the far side has no free space, and is clamped inside the edge where
 * neither has free space.
 */
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'

import { getAdjacentPlace, type Box } from '@numen/ui'
import type { Size } from '@numen/ui'
import { browserViewport, useFreeSpace, type Viewport } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
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

/* ---------------------------------- State --------------------------------- */
const root = useTemplateRef<HTMLElement>('root')

/** Its own size, which is available only from the rendered element. Placement is computed from it. */
const size = ref<Size>({ width: 0, height: 0 })

/** The area to stay inside, measured again whenever it changes. */
const { room } = useFreeSpace(props.viewport, measure)

/**
 * Horizontally, it is placed beside the item it describes. Vertically, it
 * starts where that item starts, and folds up from there at the bottom of the
 * window.
 */
const tooltipPosition = computed(() => ({
  x: getAdjacentPlace({
    from: props.at.x,
    to: props.at.x + props.at.width,
    size: size.value.width,
    room: room.value.width,
    margin: props.margin,
    gap: props.gap,
  }),
  y: getAdjacentPlace({
    from: props.at.y,
    to: props.at.y,
    size: size.value.height,
    room: room.value.height,
    margin: props.margin,
    gap: 0,
  }),
}))

const tooltipStyle = computed(() => ({
  insetInlineStart: `${tooltipPosition.value.x}px`,
  insetBlockStart: `${tooltipPosition.value.y}px`,
}))

/* ---------------------------------- Hooks --------------------------------- */
onMounted(measure)

// Measured again once the drawing has caught up with the thing it is about.
watch(() => props.at, measure, { flush: 'post' })

/* --------------------------------- Helpers -------------------------------- */
function measure(): void {
  const box = measureRect(root.value)
  if (box) size.value = { width: box.width, height: box.height }
}
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
  /* As wide as its content, whatever space remains beside it at its current position. */
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
