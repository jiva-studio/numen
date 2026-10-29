<script setup lang="ts">
/**
 * The title an edge carries, set along the line it belongs to.
 *
 * It is painted twice, with the halo finished before any letter is drawn. Each
 * glyph set along a path is a separate run, and a halo painted together with the
 * letters would overlap the adjacent letter.
 */
import type { EdgeLine } from './lines'

withDefaults(
  defineProps<{
    line: EdgeLine
    /** Drawn over the boxes, with a halo heavy enough for that. */
    isRaised?: boolean
  }>(),
  { isRaised: false },
)

const LAYERS = ['halo', 'letters'] as const
</script>

<template>
  <g
    v-if="line.titlePath"
    class="plex__edge-title"
    :class="{ 'plex__edge-title--lifted': isRaised }"
  >
    <text
      v-for="layer in LAYERS"
      :key="layer"
      class="plex__edge-label"
      :class="`plex__edge-label--${layer}`"
      :opacity="line.edge.opacity"
      text-anchor="middle"
      dominant-baseline="middle"
    >
      <textPath :href="`#${line.titlePath}`" :startOffset="line.titleAt">
        {{ line.edge.words }}
      </textPath>
    </text>
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
  fill: var(--numen-ink);
}
</style>
