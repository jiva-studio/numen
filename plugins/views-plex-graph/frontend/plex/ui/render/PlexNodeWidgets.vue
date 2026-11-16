<script setup lang="ts">
/**
 * The views of one side of a node, drawn at the node in the order of the manifest, each given the path of
 * the note. They stand as one group, centered, in a box as wide as the node; the group is handed to `hold`
 * while it is drawn, so its height can be measured.
 */
import type { PlacedNode } from '../../lib/node'
import type { PlexWidget } from '../../../types'

/** How far from the node the box reaches; the group sits against the node inside it. */
const EXTENT = 480

const props = defineProps<{
  node: PlacedNode
  side: 'above' | 'below'
  entries: readonly PlexWidget[]
  path: string
}>()

const emit = defineEmits<{
  (event: 'hold', element: Element | null): void
}>()

const hold = (element: unknown) => emit('hold', element instanceof Element ? element : null)

const getTop = () =>
  props.side === 'above'
    ? props.node.y - props.node.height / 2 - EXTENT
    : props.node.y + props.node.height / 2
</script>

<template>
  <foreignObject
    v-if="entries.length > 0"
    class="plex__widgets"
    :class="`plex__widgets--${side}`"
    :opacity="node.opacity"
    :x="node.x - node.width / 2"
    :y="getTop()"
    :width="node.width"
    :height="EXTENT"
  >
    <div class="plex__widgets-box">
      <div :ref="hold" class="plex__widgets-group">
        <component :is="entry.component" v-for="entry in entries" :key="entry.key" :path="path" />
      </div>
    </div>
  </foreignObject>
</template>

<style scoped>
.plex__widgets {
  --widgets-gap: 0.25rem;
  overflow: visible;
  pointer-events: none;
}

.plex__widgets-box {
  display: flex;
  flex-direction: column;
  block-size: 100%;
}

.plex__widgets--above .plex__widgets-box {
  justify-content: flex-end;
}

.plex__widgets-group {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: center;
  gap: var(--widgets-gap);
  pointer-events: auto;
}
</style>
