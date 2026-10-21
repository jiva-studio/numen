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

const props = defineProps<{
  /** The headings and the room they are given. */
  hung: NodeHeadings
  /** How far the box they hang from has opened, from nothing to the whole way. */
  open: number
}>()

const emit = defineEmits<{
  /** A heading was chosen. The identifier is the caller's. */
  (event: 'enter', heading: string): void
}>()

/** How far the window on the headings has been scrolled down, counted in headings. */
const scrollOffset = ref(0)

/** The headings, as far out from under the box as they have come. */
const openHeadings = computed(() => getOpenHeadings(props.hung, props.open, scrollOffset.value))

/** Where the ground under the headings begins across. */
const groundX = computed(() => props.hung.offset - props.hung.width / 2)

/** What a wheel moved that came to no whole heading, held for the next one. */
let carried = 0

// The window opens at the top each time the attention settles afresh, and is
// left at its current position while the attention leaves.
watch(
  () => props.open > 0,
  (now) => {
    if (!now) return
    scrollOffset.value = 0
    carried = 0
  },
)

/**
 * Scrolling the window over the headings. A wheel with nowhere to go is left to
 * whatever else wants it, and what it moved that came to no whole heading is
 * carried into the next one.
 */
const scrollHeadings = (event: WheelEvent) => {
  const visibleHeadings = openHeadings.value
  if (!visibleHeadings) return

  const wheel = { delta: event.deltaY, mode: event.deltaMode }
  const { by, left: remainingDelta } = scrollBy(props.hung, wheel, carried)
  if (by === 0) {
    carried = remainingDelta
    return
  }
  if (by < 0 ? !visibleHeadings.isAbove : !visibleHeadings.isBelow) {
    carried = 0
    return
  }

  carried = remainingDelta
  event.preventDefault()
  event.stopPropagation()
  // Stepped from where the window really stands, which is the picture's own
  // reckoning of it.
  scrollOffset.value = visibleHeadings.first + by
}

/** The line an arrow at an edge is drawn along. */
const arrowLine = (arrow: Arrow) =>
  arrow.points.map((at, index) => `${index === 0 ? 'M' : 'L'} ${at.x} ${at.y}`).join(' ')
</script>

<template>
  <!-- The headings are for the hand; the same headings are reached by name in the
       palette. -->
  <g v-if="openHeadings" class="plex__inside" aria-hidden="true" @wheel="scrollHeadings">
    <!-- One ground under all of them, as deep as they have come. -->
    <rect
      class="plex__ground"
      :x="groundX"
      :y="hung.top"
      :width="hung.width"
      :height="openHeadings.height"
      :opacity="openHeadings.opacity"
    />
    <PlexHeading
      v-for="heading in openHeadings.headings"
      :key="heading.id"
      :heading="heading"
      :hung="hung"
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
/* The ground the headings stand on: enough of it to hold them together, and thin
   enough to read the picture through. */
.plex__ground {
  rx: 0.25rem;
  fill: color-mix(in oklab, var(--numen-raised), transparent 25%);
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
