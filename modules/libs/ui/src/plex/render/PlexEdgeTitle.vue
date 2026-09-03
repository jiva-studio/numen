<script setup lang="ts">
/**
 * The title an edge carries, set along the line it belongs to.
 *
 * It is painted twice over, the halo finished before a letter is drawn. Each
 * glyph set along a path is a run of its own, and a halo painted with the
 * letters lies over the one beside it.
 */
import type { EdgeLine } from './lines'

withDefaults(
  defineProps<{
    line: EdgeLine
    /** Drawn over the boxes, on a halo as heavy as that asks for. */
    lifted?: boolean
  }>(),
  { lifted: false },
)

const LAYERS = ['halo', 'letters'] as const
</script>

<template>
  <g v-if="line.titlePath" class="plex__edge-title" :class="{ 'plex__edge-title--lifted': lifted }">
    <text
      v-for="layer in LAYERS"
      :key="layer"
      class="plex__edge-label"
      :class="`plex__edge-label--${layer}`"
      :opacity="line.edge.opacity"
      text-anchor="middle"
      dominant-baseline="middle"
    ><textPath
      :href="`#${line.titlePath}`"
      :startOffset="line.titleAt"
    >{{ line.edge.words }}</textPath></text>
  </g>
</template>

<style scoped>
.plex__edge-label {
  fill: var(--numen-edge-label);
  font-size: var(--numen-edge-label-size);
  stroke: var(--numen-surface);
  stroke-width: var(--numen-edge-label-halo);
  stroke-linejoin: round;
  pointer-events: none;
}

/* The halo is stroke alone; the letters over it, fill alone. */
.plex__edge-label--halo {
  fill: none;
}

.plex__edge-label--letters {
  stroke: none;
}

.plex__edge-title--lifted .plex__edge-label--halo {
  stroke-width: 6px;
}

.plex__edge-title--lifted .plex__edge-label--letters {
  fill: var(--numen-node-fg);
}
</style>
