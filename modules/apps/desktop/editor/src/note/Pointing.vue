<script setup lang="ts">
/**
 * What a link note points at, drawn over its prose: a frame playing what is at
 * the address, the words fetched for it under that, and the address itself
 * where nothing plays.
 *
 * Choosing a stretch of the words plays it. The frame is told which moment to
 * play through the messages a player takes from the page holding it, so nothing
 * that host serves runs in this window.
 */
import { computed, useTemplateRef } from 'vue'
import { clock } from '@numen/ui'
import type { Pointed } from '../core'
import type { Cue } from '../recording/transcript'

const props = defineProps<{
  points: Pointed
  cues: readonly Cue[]
  /** What the frame is called, for whoever is not looking at it. */
  words: { readonly playing: string }
}>()

const frame = useTemplateRef<HTMLIFrameElement>('frame')

// The frame is told which page holds it, so nothing else reaching this machine
// can drive the player.
const framed = computed(
  () => `${props.points.embed}&origin=${encodeURIComponent(window.location.origin)}`,
)

/** The moment a stretch of speech was said, played from there. */
const plays = (cue: Cue): void => {
  frame.value?.contentWindow?.postMessage(
    JSON.stringify({ event: 'command', func: 'seekTo', args: [cue.from / 1000, true] }),
    new URL(props.points.embed).origin,
  )
}
</script>

<template>
  <div class="pointing">
    <iframe
      v-if="props.points.embed"
      ref="frame"
      class="pointing__frame"
      :src="framed"
      :title="props.words.playing"
      sandbox="allow-scripts allow-same-origin allow-presentation"
      allow="fullscreen; picture-in-picture"
      referrerpolicy="no-referrer"
    ></iframe>

    <p v-else class="pointing__address">{{ props.points.url }}</p>

    <ol v-if="props.cues.length > 0" class="pointing__words">
      <li v-for="(cue, at) in props.cues" :key="at">
        <button type="button" class="pointing__said" @click="plays(cue)">
          <span class="pointing__at">{{ clock(cue.from) }}</span>
          {{ cue.text }}
        </button>
      </li>
    </ol>
  </div>
</template>

<style scoped>
/* What is pointed at sits at the top of the tab and keeps its height: the prose
   below it is what scrolls. */
.pointing {
  display: flex;
  flex-direction: column;
  flex: none;
  max-block-size: 60vh;
  min-block-size: 0;
  border-block-end: 1px solid var(--numen-rule);
}

/* Sixteen by nine, which is what the player inside it is drawn to. */
.pointing__frame {
  flex: none;
  display: block;
  inline-size: 100%;
  block-size: auto;
  aspect-ratio: 16 / 9;
  max-block-size: 40vh;
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

/* The words scroll under the player, and the tab's prose scrolls under them. */
.pointing__words {
  margin: 0;
  padding: var(--numen-inset) 0;
  overflow-y: auto;
  list-style: none;
  min-block-size: 0;
}

.pointing__said {
  display: block;
  inline-size: 100%;
  padding: calc(var(--numen-inset) / 2) var(--numen-gutter);
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.pointing__said:hover {
  background: var(--numen-field-border);
}

.pointing__at {
  margin-inline-end: var(--numen-inset);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
}
</style>
