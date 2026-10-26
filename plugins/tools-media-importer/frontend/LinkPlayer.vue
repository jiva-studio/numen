<script setup lang="ts">
/**
 * The copy of a video a link points at, played from this disk: a play button and how far it has
 * played.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { formatClock } from './lib/text'
import type { LinkSource } from './types'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly source: LinkSource
}>()

/* --------------------------------- State ---------------------------------- */
const media = useTemplateRef<HTMLMediaElement>('media')
const isPlaying = ref(false)
const playedMs = ref(0)
const isVideo = computed(() => props.source.mediaType.startsWith('video/'))

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  const element = media.value
  if (!element) return
  if (element.paused) void element.play()
  else element.pause()
}

function onTimeUpdate() {
  playedMs.value = (media.value?.currentTime ?? 0) * 1000
}

function onPlay() {
  isPlaying.value = true
}

function onPause() {
  isPlaying.value = false
}
</script>

<template>
  <div class="link-player">
    <video
      v-if="isVideo"
      ref="media"
      class="link-player__video"
      :src="source.mediaUrl"
      preload="metadata"
      @timeupdate="onTimeUpdate"
      @play="onPlay"
      @pause="onPause"
    />
    <audio
      v-else
      ref="media"
      :src="source.mediaUrl"
      preload="metadata"
      @timeupdate="onTimeUpdate"
      @play="onPlay"
      @pause="onPause"
    />
    <div class="link-player__controls">
      <button type="button" class="link-player__button" @click="onToggle">
        {{ isPlaying ? words.pause : words.play }}
      </button>
      <span class="link-player__time" :aria-label="words.elapsed">{{ formatClock(playedMs) }}</span>
    </div>
  </div>
</template>
