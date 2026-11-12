<script setup lang="ts">
/** The line a gesture drags behind it, and the node dropping there would leave. */
import PlexNodeView from './PlexNodeView.vue'
import type { PlacedNode } from '../../lib/node'

defineProps<{
  d: string
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
  stroke: var(--numen-edge);
  stroke-width: var(--plex-line);
  stroke-dasharray: var(--numen-thread-dash);
}
</style>
