<script setup lang="ts">
/**
 * The places inside a node, come out from under its box.
 *
 * More of them than the window holds are scrolled through with a wheel. What
 * they are and what choosing one does are the caller's.
 */
import { computed, ref, watch } from 'vue'
import { PlexHeading } from './plex-heading'
import type { NodeHeadings } from '../../lib/inside'
import { getOpenHeadings, scrollBy, type Arrow } from '../../lib/open'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  placedHeadings: NodeHeadings
  open: number
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A heading was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}>()

/* --------------------------------- State ---------------------------------- */
/** How far the window on the headings has been scrolled down, counted in headings. */
const scrollOffset = ref(0)

/** The headings, as far out from under the box as they have come. */
const openHeadings = computed(() => getOpenHeadings(props.placedHeadings, props.open, scrollOffset.value))

/** The x position where the backdrop behind the headings begins. */
const backdropX = computed(() => props.placedHeadings.offset - props.placedHeadings.width / 2)

/** What a wheel moved that came to no whole heading, held for the next one. */
let carried = 0

/* --------------------------------- Hooks ---------------------------------- */
// The window opens at the top each time the attention settles afresh, and is
// left at its current position while the attention moves away.
watch(
  () => props.open > 0,
  (now) => {
    if (!now) return
    scrollOffset.value = 0
    carried = 0
  },
)

/* -------------------------------- Handlers -------------------------------- */
const onWheel = (event: WheelEvent) => {
  const visibleHeadings = openHeadings.value
  if (!visibleHeadings) return

  const wheel = { delta: event.deltaY, mode: event.deltaMode }
  const { by, left: remainingDelta } = scrollBy(props.placedHeadings, wheel, carried)
  const direction = by !== 0 ? by : remainingDelta
  if (direction === 0) return
  if (direction < 0 ? !visibleHeadings.isAbove : !visibleHeadings.isBelow) {
    carried = 0
    return
  }

  carried = remainingDelta
  event.preventDefault()
  event.stopPropagation()
  if (by === 0) return
  // The offset steps from the first visible heading, which is the plex's own
  // calculation of the window position.
  scrollOffset.value = visibleHeadings.first + by
}

/* -------------------------------- Helpers --------------------------------- */
const arrowLine = (arrow: Arrow) =>
  arrow.points.map((pt, index) => `${index === 0 ? 'M' : 'L'} ${pt.x} ${pt.y}`).join(' ')
</script>

<template>
  <!-- The headings are for the hand; the same headings are reached by name in the
       palette. -->
  <g v-if="openHeadings" class="plex__inside" aria-hidden="true" @wheel="onWheel">
    <!-- One ground under all of them, as deep as they have come. -->
    <rect
      class="plex__backdrop"
      :x="backdropX"
      :y="placedHeadings.top"
      :width="placedHeadings.width"
      :height="openHeadings.height"
      :opacity="openHeadings.opacity"
    />
    <PlexHeading
      v-for="heading in openHeadings.headings"
      :key="heading.id"
      :heading="heading"
      :placed-headings="placedHeadings"
      @enter="emit('enter', heading.id)"
    />

    <!-- More of them than the window holds, the way they are scrolled to. -->
    <path
      v-for="arrow in openHeadings.arrows"
      :key="arrow.at"
      class="plex__more"
      :d="arrowLine(arrow)"
      :opacity="openHeadings.opacity"
    />
  </g>
</template>

<style scoped>
/* The backdrop behind the headings is nearly opaque, so the lines and names under
   it do not show through the text. */
.plex__backdrop {
  --backdrop-rx: 0.25rem;
  rx: var(--backdrop-rx);
  fill: color-mix(in oklab, var(--numen-raised), transparent 6%);
  stroke: color-mix(in oklab, var(--numen-rule), transparent 55%);
  stroke-width: var(--numen-stroke);
}

/* There is more to scroll to this way. */
.plex__more {
  fill: none;
  stroke: var(--numen-edge-label);
  stroke-width: var(--numen-stroke);
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}
</style>
