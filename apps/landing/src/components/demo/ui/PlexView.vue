<script setup lang="ts">
/**
 * The note in focus with the notes around it: parents above, children below, jumps to the left and
 * siblings to the right, joined by lines that carry the words written on the link. The notes are fixed.
 */
import { PLEX_HEIGHT, PLEX_WIDTH, PLEX_LAYOUT, type PlexNode } from '../lib/plex'
import PlexArrow from './PlexArrow.vue'
import PlexEdgeView from './PlexEdgeView.vue'

/* -------------------------------- Helpers --------------------------------- */
const getLabelId = (index: number): string => `plex-edge-${index}`
const getBoxX = (node: PlexNode): number => node.x - node.width / 2
const getBoxY = (node: PlexNode): number => node.y - node.height / 2
</script>

<template>
  <div class="plex">
    <svg class="plex__canvas" :viewBox="`0 0 ${PLEX_WIDTH} ${PLEX_HEIGHT}`" role="img" aria-label="Plex">
      <PlexArrow />

      <PlexEdgeView v-for="(edge, index) in PLEX_LAYOUT.edges" :key="edge.id" :edge="edge" :label-id="getLabelId(index)" />

      <g
        v-for="node in PLEX_LAYOUT.nodes"
        :key="node.path"
        class="plex__node"
        :data-relation="node.relation"
      >
        <rect class="plex__box" :x="getBoxX(node)" :y="getBoxY(node)" :width="node.width" :height="node.height" rx="8" />
        <text class="plex__title" :x="node.x" :y="node.y" text-anchor="middle" dominant-baseline="central">
          {{ node.text }}
        </text>
      </g>
    </svg>
  </div>
</template>

<style scoped>
.plex {
  block-size: 100%;
  inline-size: 100%;
}

.plex__canvas {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  font-family: var(--numen-font-sans);
  user-select: none;
  -webkit-user-select: none;
}

.plex__node {
  --hue: var(--numen-relation-child);
}

.plex__node[data-relation='parent'] {
  --hue: var(--numen-relation-parent);
}

.plex__node[data-relation='jump'] {
  --hue: var(--numen-relation-jump);
}

.plex__node[data-relation='sibling'] {
  --hue: var(--numen-relation-sibling);
}

.plex__box {
  fill: var(--numen-raised);
  stroke: var(--hue);
  stroke-width: var(--numen-stroke);
  transition: fill var(--numen-motion-hover) var(--numen-easing);
}

.plex__title {
  fill: var(--numen-ink);
  font-size: var(--numen-font-size);
}

.plex__node[data-relation='focus'] .plex__box {
  fill: var(--numen-accent);
  stroke: var(--numen-accent);
}

.plex__node[data-relation='focus'] .plex__title {
  fill: var(--numen-accent-ink);
}

</style>
