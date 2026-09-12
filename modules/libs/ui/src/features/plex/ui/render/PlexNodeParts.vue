<script setup lang="ts">
/**
 * The places inside a node, come out from under its box.
 *
 * More of them than the window holds are scrolled through with a wheel. What
 * they are and what choosing one does are the caller's.
 */
import { computed, ref, watch } from 'vue'
import type { HungParts } from '../../lib/inside'
import { getOpenParts, scrollBy, type Arrow } from '../../lib/open'

const props = defineProps<{
  /** The parts and the room they are given. */
  hung: HungParts
  /** How far the box they hang from has opened, from nothing to the whole way. */
  open: number
}>()

const emit = defineEmits<{
  /** A part was chosen. The identifier is the caller's. */
  (event: 'enter', part: string): void
}>()

/** How far the window on the parts has been scrolled down, counted in parts. */
const scrollOffset = ref(0)

/** The parts, as far out from under the box as they have come. */
const opened = computed(() => getOpenParts(props.hung, props.open, scrollOffset.value))

/** What a wheel moved that came to no whole part, held for the next one. */
let carried = 0

// The window opens at the top each time the attention settles afresh, and is
// left where it stands while the attention leaves.
watch(
  () => props.open > 0,
  (now) => {
    if (!now) return
    scrollOffset.value = 0
    carried = 0
  },
)

/**
 * Scrolling the window over the parts. A wheel with nowhere to go is left to
 * whatever else wants it, and what it moved that came to no whole part is
 * carried into the next one.
 */
const scrollParts = (event: WheelEvent) => {
  const shown = opened.value
  if (!shown) return

  const wheel = { delta: event.deltaY, mode: event.deltaMode }
  const { by, left } = scrollBy(props.hung, wheel, carried)
  if (by === 0) {
    carried = left
    return
  }
  if (by < 0 ? !shown.above : !shown.below) {
    carried = 0
    return
  }

  carried = left
  event.preventDefault()
  event.stopPropagation()
  // Stepped from where the window really stands, which is the picture's own
  // reckoning of it.
  scrollOffset.value = shown.first + by
}

/** The line an arrow at an edge is drawn along. */
const arrowLine = (arrow: Arrow) =>
  arrow.points.map((at, index) => `${index === 0 ? 'M' : 'L'} ${at.x} ${at.y}`).join(' ')
</script>

<template>
  <!-- The parts are for the hand; the same parts are reached by name in the
       palette. -->
  <g v-if="opened" class="plex__inside" aria-hidden="true" @wheel="scrollParts">
    <!-- One ground under all of them, as deep as they have come. -->
    <rect
      class="plex__ground"
      :x="hung.offset - hung.width / 2"
      :y="hung.top"
      :width="hung.width"
      :height="opened.height"
      :opacity="opened.opacity"
    />
    <g
      v-for="part in opened.parts"
      :key="part.id"
      :opacity="part.opacity"
      :transform="`translate(0 ${hung.top + hung.pad + part.y})`"
    >
      <foreignObject
        :x="hung.offset - hung.width / 2 + hung.pad"
        y="0"
        :width="hung.width - 2 * hung.pad"
        :height="hung.partHeight"
      >
        <div
          class="plex__part"
          :style="{
            paddingInlineStart: `calc(var(--numen-node-padding) + ${part.indent}px)`,
          }"
          @click.stop="emit('enter', part.id)"
          @dblclick.stop
        >
          <span class="plex__part-text">{{ part.text }}</span>
        </div>
      </foreignObject>
    </g>

    <!-- More of them than the window holds, the way they are scrolled to. -->
    <path
      v-for="arrow in opened.arrows"
      :key="arrow.at"
      class="plex__more"
      :d="arrowLine(arrow)"
      :opacity="opened.opacity"
    />
  </g>
</template>

<style scoped>
/* The ground the parts stand on: enough of it to hold them together, and thin
   enough to read the picture through. */
.plex__ground {
  rx: 0.25rem;
  fill: color-mix(in oklab, var(--numen-raised), transparent 25%);
  stroke: color-mix(in oklab, var(--numen-rule), transparent 55%);
  stroke-width: var(--numen-stroke);
}

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

/* There is more to scroll to this way. */
.plex__more {
  fill: none;
  stroke: var(--numen-edge-label);
  stroke-width: var(--numen-stroke);
  stroke-linecap: round;
  stroke-linejoin: round;
  pointer-events: none;
}
</style>
