<script setup lang="ts">
/**
 * The lines a gesture, a pending reach or a dragged item draws over the notes. They follow the
 * pointer, so they are drawn here and the notes around them are not redrawn with them.
 */
import { computed } from 'vue'
import PlexThread from './PlexThread.vue'
import type { PlexViewProps } from './props'
import { getDraggedShape, getPendingThread, getReachGhost, getReachThread } from './threads'
import { getRelationName } from '../../lib/relation'
import { useFieldSize } from '../../model/sizing/fieldSize'

const props = withDefaults(
  defineProps<
    Pick<
      PlexViewProps,
      | 'frame'
      | 'viewport'
      | 'ghostSize'
      | 'relationName'
      | 'gesture'
      | 'dragging'
      | 'pending'
      | 'dropName'
      | 'curvature'
      | 'showPills'
    >
  >(),
  {
    relationName: getRelationName,
    pending: null,
    dropName: getRelationName,
    showPills: true,
  },
)

const field = useFieldSize()

const fieldSize = computed(() => props.ghostSize ?? field.value)

const threadOptions = computed(() => ({
  curvature: props.curvature,
  hasPill: props.showPills,
  viewportWidth: props.viewport.width,
}))

const thread = computed(() =>
  getReachThread(
    props.frame,
    props.gesture?.from.value ?? null,
    props.gesture?.at.value ?? null,
    threadOptions.value,
  ),
)

const pendingThread = computed(() =>
  getPendingThread(props.frame, {
    from: props.pending?.from ?? null,
    position: props.pending?.at ?? null,
    viewport: props.viewport,
    ...threadOptions.value,
  }),
)

const ghost = computed(() =>
  getReachGhost(props.gesture?.outcome.value ?? null, props.gesture?.at.value ?? null, {
    relationName: props.relationName,
    nodeSize: fieldSize.value,
  }),
)

const dragging = computed(() =>
  getDraggedShape(
    props.frame,
    { relation: props.dragging?.relation.value ?? null, to: props.dragging?.at.value ?? null },
    {
      dropName: props.dropName,
      nodeSize: fieldSize.value,
      curvature: props.curvature,
    },
  ),
)
</script>

<template>
  <PlexThread v-if="thread" class="plex__reach" :d="thread" :ghost="ghost" />

  <PlexThread v-if="pendingThread" class="plex__pending" :d="pendingThread" />

  <PlexThread v-if="dragging" class="plex__dragged" :d="dragging.thread" :ghost="dragging.ghost" />
</template>

<style scoped>
.plex__reach,
.plex__pending,
.plex__dragged {
  pointer-events: none;
}
</style>
