<script setup lang="ts">
/**
 * The box of one node and the title standing in it.
 *
 * The title goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import { Comment, computed, Fragment, Text, useSlots, type VNode } from 'vue'
import type { WideBox } from '../../model/sizing/expansion'
import type { PlacedNode } from '../../lib/node'

const props = defineProps<{
  node: PlacedNode
  box: WideBox
  isGhost: boolean
  isTarget: boolean
  showPills: boolean
}>()

defineSlots<{
  /** What is drawn beside this node's title. */
  icon?(props: { node: PlacedNode }): VNode[]
}>()

const slots = useSlots()

/**
 * Whether this node is drawn something before its title.
 */
const hasIcon = computed(() => hasAnything(slots.icon?.({ node: props.node })))

/** Where the box begins, which everything drawn in it is placed from. */
const startsAt = computed(() => props.box.offset - props.box.width / 2)

/** Where the box begins down, which is half its height above the middle. */
const startsDown = computed(() => -props.node.height / 2)

/** A ghost keeps its dashed outline whether or not the other nodes have pills. */
const hasPill = computed(() => props.showPills || props.isGhost)

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
    v-if="hasPill"
    class="plex__box"
    :x="startsAt"
    :y="startsDown"
    :width="box.width"
    :height="node.height"
  />
  <template v-else>
    <!-- Without a pill the node still answers the pointer over the whole of its box. -->
    <rect
      class="plex__area"
      :x="startsAt"
      :y="startsDown"
      :width="box.width"
      :height="node.height"
    />
    <rect
      v-if="isTarget"
      class="plex__ring"
      :x="startsAt"
      :y="startsDown"
      :width="box.width"
      :height="node.height"
    />
  </template>
  <foreignObject :x="startsAt" :y="startsDown" :width="box.width" :height="node.height">
    <div class="plex__title" :class="{ 'caps-numen': $host, 'plex__title--bare': !hasPill }">
      <span v-if="!hasPill" class="plex__dot" aria-hidden="true" />
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
/* The hue comes from the node's own relation, so a new relation needs a token and
   nothing here. The outline changes over the length of the move that changes
   the relation; the fill answers the pointer at the speed a pointer is answered. */
.plex__box {
  rx: var(--radius);
  fill: var(--numen-raised);
  stroke: var(--relation-hue);
  stroke-width: var(--plex-outline, var(--numen-stroke));
  transition:
    fill var(--numen-motion-hover) var(--numen-easing),
    stroke var(--plex-move) var(--numen-easing);
}

/* Icon then title, centered together in a box of a size the arrangement chose. A
   title is something to look at and press, and takes no selection. */
.plex__title {
  block-size: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--numen-node-gap);
  padding-inline: var(--plex-pad-inline, var(--numen-node-padding));
  box-sizing: border-box;
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
  line-height: var(--numen-line-height);
  pointer-events: none;
  user-select: none;
  -webkit-user-select: none;
  transition: color var(--plex-move) var(--numen-easing);
}

/* The area is filled with nothing, which still takes the pointer: `fill: none` would not. */
.plex__area {
  fill: transparent;
}

.plex__ring {
  rx: var(--radius);
  fill: none;
  stroke: var(--numen-ring);
  stroke-width: var(--numen-ring-width);
}

.plex__dot {
  flex: none;
  inline-size: var(--plex-dot, 0.375rem);
  block-size: var(--plex-dot, 0.375rem);
  border-radius: 50%;
  background: var(--relation-hue);
}

.plex__icon {
  flex: none;
  display: flex;
  align-items: center;
  color: var(--relation-hue);
}

.plex__title-text {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
