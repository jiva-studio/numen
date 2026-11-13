<script setup lang="ts">
/**
 * What a node is made of in the picture: its box and title, the headings that come out from
 * under it, and the handle to reach out from. A node draws it in its own place, or in the layer
 * over the others while it is open.
 */
import PlexNodeBox from './PlexNodeBox.vue'
import PlexNodeHandle from './PlexNodeHandle.vue'
import PlexNodeHeadings from './PlexNodeHeadings.vue'
import type { PlexDrawnSlots } from './props'
import type { NodeHeadings } from '../../lib/inside'
import type { PlacedNode, Position } from '../../lib/node'
import type { WideBox } from '../../model/sizing/expansion'

defineProps<{
  node: PlacedNode
  box: WideBox
  isGhost: boolean
  isTarget: boolean
  showPills: boolean
  restHeight: number
  hung: NodeHeadings | null
  open: number
  isHandleShown: boolean
  handle: Position
}>()

const emit = defineEmits<{
  (event: 'enter', heading: string): void
  (event: 'reach', pointer: PointerEvent): void
  (event: 'ask'): void
}>()

defineSlots<PlexDrawnSlots>()
</script>

<template>
  <PlexNodeBox
    :node="node"
    :box="box"
    :is-ghost="isGhost"
    :is-target="isTarget"
    :show-pills="showPills"
    :rest-height="restHeight"
  >
    <template v-if="$slots.icon" #icon="{ node: drawn }">
      <slot name="icon" :node="drawn" />
    </template>
  </PlexNodeBox>

  <PlexNodeHeadings
    v-if="hung"
    :hung="hung"
    :open="open"
    @enter="(heading) => emit('enter', heading)"
  />

  <!-- Reach out from here to make something. Under the hand or under the
       keyboard, so it is there when wanted and out of the way when not. -->
  <PlexNodeHandle
    v-if="isHandleShown"
    :at="handle"
    @reach="(pointer) => emit('reach', pointer)"
    @ask="emit('ask')"
  />
</template>
