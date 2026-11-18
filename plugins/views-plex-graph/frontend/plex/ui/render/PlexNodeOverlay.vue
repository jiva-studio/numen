<script setup lang="ts">
/**
 * The views of the slot across a node, each drawn over the whole box of the node and free to reach beyond it:
 * the box they are drawn in is the whole surface. Each view is given the path of the note, the middle of the
 * node (`x`, `y`), its size and how many times larger than a note at rest it is drawn (`scale`), in the units
 * of the surface, and follows the node as it moves. The surface takes no
 * pointer events: a view takes those of the elements it sets `pointer-events` on.
 */
import { computed } from 'vue'
import { getScale, type PlacedNode } from '../../lib/node'
import type { PlexWidget } from '../../../types'

const props = defineProps<{
  node: PlacedNode
  entries: readonly PlexWidget[]
  path: string
  restHeight: number
  viewport: { width: number; height: number }
}>()

const scale = computed(() => getScale(props.node.height, props.restHeight))

const boxStyle = computed(() => ({
  left: `${props.viewport.width / 2 + props.node.x - props.node.width / 2}px`,
  top: `${props.viewport.height / 2 + props.node.y - props.node.height / 2}px`,
  width: `${props.node.width}px`,
  height: `${props.node.height}px`,
}))
</script>

<template>
  <foreignObject
    class="plex__over"
    :x="-viewport.width / 2"
    :y="-viewport.height / 2"
    :width="viewport.width"
    :height="viewport.height"
  >
    <div class="plex__over-box" :style="boxStyle">
      <div
        v-for="entry in entries"
        :key="entry.key"
        class="plex__over-view"
        :data-plex-widget="entry.key"
      >
        <component
          :is="entry.component"
          :path="path"
          :x="node.x"
          :y="node.y"
          :width="node.width"
          :height="node.height"
          :scale="scale"
        />
      </div>
    </div>
  </foreignObject>
</template>

<style scoped>
.plex__over {
  overflow: visible;
  pointer-events: none;
}

.plex__over-box {
  position: absolute;
}

.plex__over-view {
  position: absolute;
  inset: 0;
}
</style>
