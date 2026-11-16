<script setup lang="ts">
/**
 * The views of one side of a node, drawn at the node in the order the loader sorts them (the slot `order`,
 * then the plugin id), each in its own centered row and given the path of the note. The rows are one column,
 * the group; its element is emitted so its height can be measured.
 * The group holds the `--numen-node-gap` against the node as padding, so the measured height includes it.
 */
import type { ComponentPublicInstance } from 'vue'
import type { PlacedNode } from '../../lib/node'
import type { PlexWidget } from '../../../types'

const props = defineProps<{
  node: PlacedNode
  side: 'above' | 'below'
  entries: readonly PlexWidget[]
  path: string
}>()

const emit = defineEmits<{
  (event: 'widgets', element: Element | null): void
}>()

/** How far from the node the box reaches; the group sits against the node inside it. */
const EXTENT = 480

const onWidgetsChange = (element: Element | ComponentPublicInstance | null) =>
  emit('widgets', element instanceof Element ? element : null)

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
      <div :ref="onWidgetsChange" class="plex__widgets-group">
        <div
          v-for="entry in entries"
          :key="entry.key"
          class="plex__widget"
          :data-plex-widget="entry.key"
        >
          <component :is="entry.component" :path="path" />
        </div>
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
  flex-direction: column;
  align-items: center;
  gap: var(--widgets-gap);
  pointer-events: auto;
}

.plex__widget {
  display: flex;
  justify-content: center;
  max-inline-size: 100%;
}

.plex__widgets--above .plex__widgets-group {
  padding-block-end: var(--numen-node-gap);
}

.plex__widgets--below .plex__widgets-group {
  padding-block-start: var(--numen-node-gap);
}
</style>
