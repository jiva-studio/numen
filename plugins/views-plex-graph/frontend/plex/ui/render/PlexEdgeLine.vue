<script setup lang="ts">
/** One edge: the curve, and the arrowheads it ends in where it carries any. */
import { ARROWHEAD_PATH } from '../../lib/arrange'
import type { EdgeLine } from './lines'

withDefaults(
  defineProps<{
    line: EdgeLine
    /** Drawn over the boxes, in the colours that stand out against them. */
    isRaised?: boolean
  }>(),
  { isRaised: false },
)
</script>

<template>
  <g class="plex__edge-line" :class="{ 'plex__edge-line--lifted': isRaised }">
    <path class="plex__edge" :d="line.d" :opacity="line.edge.opacity" />
    <path
      v-for="arrow in line.arrows"
      :key="arrow.end"
      class="plex__edge-arrow"
      :d="ARROWHEAD_PATH"
      :transform="arrow.transform"
      :opacity="line.edge.opacity"
    />
  </g>
</template>

<style scoped>
.plex__edge {
  fill: none;
  stroke: var(--numen-edge);
  stroke-width: var(--numen-edge-width);
  stroke-linecap: round;
}

/* The head, drawn about its own tip and turned onto its end of the line by the
   transform it is given. It stays within the room the arrangement keeps for it
   at the end of a line. */
.plex__edge-arrow {
  fill: var(--numen-edge);
}

.plex__edge-line--lifted .plex__edge {
  stroke: color-mix(in oklab, var(--numen-edge), var(--numen-ink) 55%);
}

.plex__edge-line--lifted .plex__edge-arrow {
  fill: color-mix(in oklab, var(--numen-edge), var(--numen-ink) 55%);
}
</style>
