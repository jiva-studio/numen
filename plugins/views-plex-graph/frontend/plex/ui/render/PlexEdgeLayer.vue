<script setup lang="ts">
/**
 * One layer of the lines: the resting lines, the titles of the resting lines, or the raised lines
 * with their titles. It reads which lines are raised, so a change of attention redraws this layer
 * and not the drawing around it.
 */
import PlexEdgeLine from './PlexEdgeLine.vue'
import PlexEdgeTitle from './PlexEdgeTitle.vue'
import { useLayerLines, type EdgeLine } from './lines'

const props = defineProps<{
  lines: readonly EdgeLine[]
  isRaised: (line: EdgeLine) => boolean
  layer: 'resting' | 'titles' | 'raised'
  showEdgeLabels: boolean
}>()

const { raised, resting } = useLayerLines(() => props)
</script>

<template>
  <g v-if="layer === 'resting'" aria-hidden="true">
    <PlexEdgeLine v-for="line in resting" :key="line.key" :line="line" />
  </g>

  <g v-else-if="layer === 'titles'" aria-hidden="true">
    <PlexEdgeTitle v-for="line in resting" :key="`title:${line.key}`" :line="line" />
  </g>

  <g v-else-if="raised.length" class="plex__raised" aria-hidden="true">
    <template v-for="line in raised" :key="line.key">
      <PlexEdgeLine :line="line" is-raised />
      <PlexEdgeTitle v-if="showEdgeLabels" :line="line" is-raised />
    </template>
  </g>
</template>

<style scoped>
.plex__raised {
  pointer-events: none;
}
</style>
