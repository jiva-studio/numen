<script setup lang="ts">
/**
 * One place inside a node, drawn in a box of its own on the ground the headings
 * stand on.
 */
import { computed } from 'vue'
import type { NodeHeadings } from '../../../lib/inside'
import type { DrawnHeading } from '../../../lib/open'

const props = defineProps<{
  /** The heading, as far out from under the box as it has come. */
  heading: DrawnHeading
  /** The headings and the room they are given. */
  hung: NodeHeadings
}>()

const emit = defineEmits<{
  /** The heading was chosen. The identifier is the caller's. */
  (event: 'enter'): void
}>()

/** Where the heading goes, and the box it is drawn in. */
const place = computed(() => ({
  transform: `translate(0 ${props.hung.top + props.hung.pad + props.heading.y})`,
  x: props.hung.offset - props.hung.width / 2 + props.hung.pad,
  width: props.hung.width - 2 * props.hung.pad,
}))

const headingStyle = computed(() => ({
  paddingInlineStart: `calc(var(--numen-node-padding) + ${props.heading.indent}px)`,
}))
</script>

<template>
  <g :opacity="heading.opacity" :transform="place.transform">
    <foreignObject :x="place.x" y="0" :width="place.width" :height="hung.headingHeight">
      <div class="plex__heading" :style="headingStyle" @click.stop="emit('enter')" @dblclick.stop>
        <span class="plex__heading-text">{{ heading.text }}</span>
      </div>
    </foreignObject>
  </g>
</template>

<style scoped>
/* Each heading is drawn in a box of its own, and where that box goes and how far
   it has faded up are SVG attributes on the group holding it. HTML inside a
   `foreignObject` that takes a layer of its own — under `opacity`, under
   `transform` — is drawn at the page's origin in WebKit, which is the engine
   the window is drawn in. */
.plex__heading {
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
.plex__heading:hover {
  background: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 12%);
}

/* One line, then an ellipsis, as a title is. */
.plex__heading-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
