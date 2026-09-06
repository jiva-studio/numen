<script setup lang="ts">
/**
 * What a link note points at, drawn over its prose.
 *
 * An address something plays is played in a frame. An address nothing plays is
 * the address itself, which is what the person pasted and what a browser opens.
 */
import { computed } from 'vue'
import type { Pointed } from '../core'

const props = defineProps<{
  points: Pointed
  /** What the frame is called, for whoever is not looking at it. */
  words: { readonly playing: string }
}>()

// The frame is told which page holds it, so nothing else reaching this machine
// can drive the player.
const framed = computed(
  () => `${props.points.embed}&origin=${encodeURIComponent(window.location.origin)}`,
)
</script>

<template>
  <div class="pointing">
    <iframe
      v-if="props.points.embed"
      class="pointing__frame"
      :src="framed"
      :title="props.words.playing"
      sandbox="allow-scripts allow-same-origin allow-presentation"
      allow="fullscreen; picture-in-picture"
      referrerpolicy="no-referrer"
    ></iframe>

    <p v-else class="pointing__address">{{ props.points.url }}</p>
  </div>
</template>

<style scoped>
/* What is pointed at sits at the top of the tab and keeps its height: the prose
   below it is what scrolls. */
.pointing {
  flex: none;
  border-block-end: 1px solid var(--numen-rule);
}

/* Sixteen by nine, which is what the player inside it is drawn to. */
.pointing__frame {
  display: block;
  inline-size: 100%;
  block-size: auto;
  aspect-ratio: 16 / 9;
  max-block-size: 45vh;
  border: 0;
  background: #000;
}

.pointing__address {
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  overflow-wrap: anywhere;
}
</style>
