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
import type { Address } from '../core'
import type { Cue } from '../recording/transcript'

const props = defineProps<{
  address: Address
  cues: readonly Cue[]
  /**
   * Where a copy of it on this disk is played from, and nothing where there is
   * none. A copy is played in place of the frame: it plays offline, and nothing
   * of the site it came from is loaded to play it.
   */
  copy: string
  /** What the frame is called, for whoever is not looking at it. */
  words: { readonly playing: string }
}>()

const frame = useTemplateRef<HTMLIFrameElement>('frame')
const player = useTemplateRef<HTMLVideoElement>('player')

// The frame is told which page holds it, so nothing else reaching this machine
// can drive the player.
const framed = computed(
  () => `${props.address.embed}&origin=${encodeURIComponent(window.location.origin)}`,
)

/**
 * The moment a stretch of speech was said, played from there — in the copy on
 * this disk where there is one, and in the frame otherwise.
 */
const plays = (cue: Cue): void => {
  if (player.value) {
    player.value.currentTime = cue.from / 1000
    void player.value.play()
    return
  }
  frame.value?.contentWindow?.postMessage(
    JSON.stringify({ event: 'command', func: 'seekTo', args: [cue.from / 1000, true] }),
    new URL(props.address.embed).origin,
  )
}
</script>

<template>
  <div class="embed">
    <video
      v-if="props.copy"
      ref="player"
      class="embed__frame"
      :src="props.copy"
      :title="props.words.playing"
      controls
      preload="metadata"
    ></video>

    <iframe
      v-else-if="props.address.embed"
      ref="frame"
      class="embed__frame"
      :src="framed"
      :title="props.words.playing"
      sandbox="allow-scripts allow-same-origin allow-presentation"
      allow="fullscreen; picture-in-picture"
      referrerpolicy="no-referrer"
    ></iframe>

    <p v-else class="embed__address">{{ props.address.url }}</p>

    <ol v-if="props.cues.length > 0" class="embed__words">
      <li v-for="(cue, at) in props.cues" :key="at">
        <button type="button" class="embed__said" @click="plays(cue)">
          <span class="embed__at">{{ clock(cue.from) }}</span>
          {{ cue.text }}
        </button>
      </li>
    </ol>
  </div>
</template>

<style scoped>
/* What is at the address sits at the top of the tab and keeps its height: the
   prose below it is what scrolls. */
.embed {
  display: flex;
  flex-direction: column;
  flex: none;
  max-block-size: 60vh;
  min-block-size: 0;
  border-block-end: 1px solid var(--numen-rule);
}

/* Sixteen by nine, which is what the player inside it is drawn to. */
.embed__frame {
  flex: none;
  display: block;
  inline-size: 100%;
  block-size: auto;
  aspect-ratio: 16 / 9;
  max-block-size: 40vh;
  border: 0;
  background: #000;
}

.embed__address {
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  overflow-wrap: anywhere;
}

/* The words scroll under the player, and the tab's prose scrolls under them. */
.embed__words {
  margin: 0;
  padding: var(--numen-inset) 0;
  overflow-y: auto;
  list-style: none;
  min-block-size: 0;
}

.embed__said {
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

.embed__said:hover {
  background: var(--numen-field-border);
}

.embed__at {
  margin-inline-end: var(--numen-inset);
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
}
</style>
