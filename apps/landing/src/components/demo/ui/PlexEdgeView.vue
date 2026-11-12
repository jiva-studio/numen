<script setup lang="ts">
/** A line between two notes, with the words written on the link along it. */
import type { PlexEdge } from '../lib/plex'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  edge: PlexEdge
  labelId: string
}>()
</script>

<template>
  <g>
    <path class="plex__edge" :d="edge.path" marker-end="url(#plex-arrow)" />
    <path :id="labelId" class="plex__guide" :d="edge.labelPath" />
    <text v-if="edge.label" class="plex__label" text-anchor="middle" dy="-3">
      <textPath :href="`#${labelId}`" startOffset="50%">{{ edge.label }}</textPath>
    </text>
  </g>
</template>

<style scoped>
.plex__edge {
  fill: none;
  stroke: var(--numen-edge);
  stroke-width: var(--numen-edge-width);
  stroke-linecap: round;
}

.plex__guide {
  fill: none;
  stroke: none;
}

.plex__label {
  fill: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  stroke: var(--numen-surface);
  stroke-width: var(--numen-edge-label-halo);
  stroke-linejoin: round;
  paint-order: stroke;
  pointer-events: none;
}
</style>
