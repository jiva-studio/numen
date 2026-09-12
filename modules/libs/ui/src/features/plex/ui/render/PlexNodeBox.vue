<script setup lang="ts">
/**
 * The box of one node and the title standing in it.
 *
 * The title goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import { Comment, computed, Fragment, Text, useSlots, type VNode } from 'vue'
import type { WideBox } from '../../model/dwell'
import type { PlacedNode } from '../../lib/node'

const props = defineProps<{
  node: PlacedNode
  /** The box as it is drawn: its width, and how far its middle has slid. */
  box: WideBox
  /** Not a node yet, so its title is the seat it would take. */
  ghost: boolean
}>()

defineSlots<{
  /** What is drawn beside this node's title. */
  icon?(props: { node: PlacedNode }): unknown
}>()

const slots = useSlots()

/**
 * Whether this node is drawn something before its title.
 */
const hasIcon = computed(() => hasAnything(slots.icon?.({ node: props.node })))

/** Where the box begins, which everything drawn in it is placed from. */
const startsAt = computed(() => props.box.offset - props.box.width / 2)

/** Whether anything was drawn at all, which a placeholder and a blank are not. */
function hasAnything(vnodes: readonly VNode[] | undefined): boolean {
  return (
    !!vnodes &&
    vnodes.some((one) => {
      if (one.type === Comment) return false
      if (one.type === Fragment) return hasAnything(one.children as VNode[])
      if (one.type === Text) return String(one.children).trim() !== ''
      return true
    })
  )
}
</script>

<template>
  <rect
    class="plex__box"
    :x="startsAt"
    :y="-node.height / 2"
    :width="box.width"
    :height="node.height"
  />
  <foreignObject
    :x="startsAt"
    :y="-node.height / 2"
    :width="box.width"
    :height="node.height"
  >
    <div class="plex__title" :class="{ 'caps-numen': ghost }">
      <!-- Whatever stands for the thing a node addresses. The plex has no
           way to know what that is, so it is handed one. -->
      <span v-if="hasIcon" class="plex__icon" aria-hidden="true">
        <slot name="icon" :node="node" />
      </span>
      <span class="plex__title-text">{{ node.title }}</span>
    </div>
  </foreignObject>
</template>

<style scoped>
/* The hue comes from the node's own seat, so a new seat needs a token and
   nothing here. The outline changes over the length of the move that changes
   the seat; the fill answers the pointer at the speed a pointer is answered. */
.plex__box {
  rx: var(--radius);
  fill: var(--numen-raised);
  stroke: var(--numen-seat-hue, var(--numen-rule));
  stroke-width: var(--numen-stroke);
  transition:
    fill var(--numen-motion-hover) var(--numen-easing),
    stroke var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

/* Icon then title, centred together in a box of a size the arrangement chose. A
   title is something to look at and press, and takes no selection. */
.plex__title {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  padding-inline: var(--numen-node-padding);
  box-sizing: border-box;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  user-select: none;
  -webkit-user-select: none;
  transition: color var(--numen-plex-move, var(--numen-motion)) var(--numen-easing);
}

.plex__icon {
  flex: none;
  display: flex;
  align-items: center;
  color: var(--numen-seat-hue, var(--numen-ink));
}

/* One line, then an ellipsis. A box stands at the height the arrangement gave
   it, whatever its title runs to. */
.plex__title-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
