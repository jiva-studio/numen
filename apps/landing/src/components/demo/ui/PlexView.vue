<script setup lang="ts">
/**
 * The note in focus with the notes around it: parents above, children below, jumps to the left and
 * siblings to the right, joined by lines that carry the words written on the link.
 */
import { ref } from 'vue'
import { Menu, type Position } from '@numen/ui'
import { PLEX_MENU } from '../lib/menus'
import { PLEX_HEIGHT, PLEX_WIDTH, type PlexLayout, type PlexNode } from '../lib/plexLayout'
import { getCommandIcon } from './icons'
import PlexArrow from './PlexArrow.vue'
import PlexEdgeView from './PlexEdgeView.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ layout: PlexLayout }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  go: [path: string]
  open: [path: string]
  run: [command: string, path: string]
}>()

/* --------------------------------- State ---------------------------------- */
const menu = ref<{ path: string; at: Position } | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpenMenu(event: MouseEvent, node: PlexNode) {
  menu.value = { path: node.path, at: { x: event.clientX, y: event.clientY } }
}

function onChooseItem(id: string) {
  if (menu.value) emit('run', id, menu.value.path)
  menu.value = null
}

/* -------------------------------- Helpers --------------------------------- */
const getLabelId = (index: number): string => `plex-edge-${index}`
const getBoxX = (node: PlexNode): number => node.x - node.width / 2
const getBoxY = (node: PlexNode): number => node.y - node.height / 2
</script>

<template>
  <div class="plex">
    <svg class="plex__canvas" :viewBox="`0 0 ${PLEX_WIDTH} ${PLEX_HEIGHT}`" role="img" aria-label="Plex">
      <PlexArrow />

      <PlexEdgeView v-for="(edge, index) in layout.edges" :key="edge.id" :edge="edge" :label-id="getLabelId(index)" />

      <g
        v-for="node in layout.nodes"
        :key="node.path"
        class="plex__node"
        :data-relation="node.relation"
        @click="emit('go', node.path)"
        @dblclick="emit('open', node.path)"
        @contextmenu.prevent="onOpenMenu($event, node)"
      >
        <rect class="plex__box" :x="getBoxX(node)" :y="getBoxY(node)" :width="node.width" :height="node.height" rx="8" />
        <text class="plex__title" :x="node.x" :y="node.y" text-anchor="middle" dominant-baseline="central">
          {{ node.text }}
        </text>
      </g>
    </svg>

    <Menu v-if="menu" :items="PLEX_MENU" :at="menu.at" is-open @choose="onChooseItem" @dismiss="menu = null">
      <template #icon="{ id }">
        <component :is="getCommandIcon(id)" v-if="getCommandIcon(id)" class="plex__icon" aria-hidden="true" />
      </template>
    </Menu>
  </div>
</template>

<style scoped>
.plex {
  block-size: 100%;
  inline-size: 100%;
}

.plex__canvas {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  font-family: var(--numen-font-sans);
  user-select: none;
  -webkit-user-select: none;
}

.plex__node {
  cursor: pointer;
  --hue: var(--numen-relation-child);
}

.plex__node[data-relation='parent'] {
  --hue: var(--numen-relation-parent);
}

.plex__node[data-relation='jump'] {
  --hue: var(--numen-relation-jump);
}

.plex__node[data-relation='sibling'] {
  --hue: var(--numen-relation-sibling);
}

.plex__box {
  fill: var(--numen-raised);
  stroke: var(--hue);
  stroke-width: var(--numen-stroke);
  transition: fill var(--numen-motion-hover) var(--numen-easing);
}

.plex__node:hover .plex__box {
  fill: color-mix(in oklab, var(--numen-raised), var(--hue) 10%);
}

.plex__title {
  fill: var(--numen-ink);
  font-size: var(--numen-font-size);
}

.plex__node[data-relation='focus'] .plex__box {
  fill: var(--numen-accent);
  stroke: var(--numen-accent);
}

.plex__node[data-relation='focus'] .plex__title {
  fill: var(--numen-accent-ink);
}

.plex__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
