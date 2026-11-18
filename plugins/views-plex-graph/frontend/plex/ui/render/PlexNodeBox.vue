<script setup lang="ts">
/**
 * The box of one node and the title standing in it.
 *
 * The title goes through a `foreignObject`: SVG text cannot ellipsise and does
 * not reorder a right-to-left run.
 */
import { Comment, computed, Fragment, Text, useSlots, type VNode } from 'vue'
import type { ExpandedBox } from '../../model/sizing/expansion'
import { getScale, type PlacedNode } from '../../lib/node'

const props = withDefaults(
  defineProps<{
    node: PlacedNode
    box: ExpandedBox
    isGhost: boolean
    isTarget: boolean
    showPills: boolean
    restHeight?: number
  }>(),
  { restHeight: 0 },
)

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

/**
 * How much larger the title is drawn: as much as the node is taller than a note at rest, so the title grows with
 * its box in every mode, and through the move to focus it keeps the same room in the box.
 * The title is laid out in a box that many times smaller and enlarged, so text, icon, gap and padding grow together.
 */
const scale = computed(() => (props.isGhost ? 1 : getScale(props.node.height, props.restHeight)))

/**
 * The room the title is laid out in, in the units the title is drawn in. A node in a movement lays its
 * title out in the size it settles with, so the box changing size does not cut the title again.
 * A box opened wider than the node takes the whole of it.
 */
const titleRoom = computed(() => {
  const { node, box, restHeight, isGhost } = props
  if (!node.settled || box.width > node.width) return box.width / scale.value
  const settledScale = isGhost ? 1 : getScale(node.settled.height, restHeight)
  return node.settled.width / settledScale
})

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
  <foreignObject
    :x="box.offset / scale - titleRoom / 2"
    :y="startsDown / scale"
    :width="titleRoom"
    :height="node.height / scale"
    :transform="scale === 1 ? undefined : `scale(${scale})`"
  >
    <div class="plex__title" :class="{ 'caps-numen': $host, 'plex__title--bare': !hasPill }">
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
  stroke-width: var(--plex-outline);
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
  padding-inline: var(--plex-pad-inline);
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
