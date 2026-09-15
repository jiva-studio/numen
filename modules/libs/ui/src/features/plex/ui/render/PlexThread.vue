<script setup lang="ts">
/** The line a gesture drags behind it, and the node letting go there would leave. */
import PlexNodeView from './PlexNodeView.vue'
import type { PlacedNode } from '../../lib/node'

defineProps<{
  /** The line itself, as a path. */
  d: string
  /** The shape letting go would leave, and nothing where it would leave none. */
  ghost?: PlacedNode | null
}>()
</script>

<template>
  <g>
    <path class="plex__thread" :d="d" aria-hidden="true" />
    <PlexNodeView v-if="ghost" :node="ghost" gesture-role="ghost" />
  </g>
</template>

<style scoped>
.plex__thread {
  fill: none;
  stroke: var(--numen-ring);
  stroke-width: var(--numen-edge-width);
  stroke-dasharray: var(--numen-thread-dash);
}
</style>
