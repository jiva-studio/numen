<script setup lang="ts">
/**
 * One place inside a node, drawn in a box of its own on the ground the parts
 * stand on.
 */
import { computed } from 'vue'
import type { NodeParts } from '../../../lib/inside'
import type { DrawnPart } from '../../../lib/open'

const props = defineProps<{
  /** The part, as far out from under the box as it has come. */
  part: DrawnPart
  /** The parts and the room they are given. */
  hung: NodeParts
}>()

const emit = defineEmits<{
  /** The part was chosen. The identifier is the caller's. */
  (event: 'enter'): void
}>()

const partStyle = computed(() => ({
  paddingInlineStart: `calc(var(--numen-node-padding) + ${props.part.indent}px)`,
}))
</script>

<template>
  <g :opacity="part.opacity" :transform="`translate(0 ${hung.top + hung.pad + part.y})`">
    <foreignObject
      :x="hung.offset - hung.width / 2 + hung.pad"
      y="0"
      :width="hung.width - 2 * hung.pad"
      :height="hung.partHeight"
    >
      <div class="plex__part" :style="partStyle" @click.stop="emit('enter')" @dblclick.stop>
        <span class="plex__part-text">{{ part.text }}</span>
      </div>
    </foreignObject>
  </g>
</template>

<style scoped>
/* Each part is drawn in a box of its own, and where that box goes and how far
   it has faded up are SVG attributes on the group holding it. HTML inside a
   `foreignObject` that takes a layer of its own — under `opacity`, under
   `transform` — is drawn at the page's origin in WebKit, which is the engine
   the window is drawn in. */
.plex__part {
  block-size: 100%;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-edge-label-size);
  color: var(--numen-ink);
  user-select: none;
  -webkit-user-select: none;
  display: flex;
  align-items: center;
  box-sizing: border-box;
  padding-inline-end: var(--numen-node-padding);
  border-radius: 0.1875rem;
  cursor: pointer;
  transition: background var(--numen-motion-hover) var(--numen-easing);
}

/* A ground under the one the hand is on, which is what says it can be pressed. */
.plex__part:hover {
  background: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 12%);
}

/* One line, then an ellipsis, as a title is. */
.plex__part-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
