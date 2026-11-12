<script setup lang="ts">
/**
 * What a url points at, played at the top of its tab: a copy on this disk where
 * there is one, and a frame where there is not.
 *
 * The transcript stands under this and is where a moment is chosen. What that
 * moment does is here: a copy is seeked, and a frame is told through the
 * messages a player takes from the page holding it, so nothing that host serves
 * runs in this window.
 */
import { computed, useTemplateRef } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'
import { getFrameOrigin, LEAST, readPlayerInfo, seekFrame } from '../lib/frame'
import type { HeightStore } from '../model/heightStore'
import { useWindowMessage } from '../model/useWindowMessage'
import { useFrameHeight } from './useFrameHeight'

const props = defineProps<{
  embed: string
  copy: string
  words: { readonly playing: string; readonly taller: string }
  store?: HeightStore
}>()

const emit = defineEmits<{
  'time-update': [ms: number]
  'play-state': [isPlaying: boolean]
}>()

defineExpose({ seek })

const root = useTemplateRef<HTMLElement>('root')
const frame = useTemplateRef<HTMLIFrameElement>('frame')
const player = useTemplateRef<HTMLVideoElement>('player')
const media = computed(() => player.value ?? frame.value)
const { tall, isHeld, tallest, handleValue, frameStyle, onPointerDown, onHandleKeydown } =
  useFrameHeight(props.store, root, media)

useWindowMessage(onWindowMessage)

function onVideoTimeUpdate(event: Event): void {
  const video = event.target as HTMLVideoElement
  emit('time-update', Math.round(video.currentTime * 1000))
}

function onFrameLoad(): void {
  const origin = getFrameOrigin(props.embed)
  if (origin === null) return
  frame.value?.contentWindow?.postMessage(
    JSON.stringify({ event: 'listening', id: 1, channel: 'widget' }),
    origin,
  )
}

function onWindowMessage(event: MessageEvent): void {
  if (!frame.value || event.source !== frame.value.contentWindow) return
  const expectedOrigin = getFrameOrigin(props.embed)
  if (!expectedOrigin || event.origin !== expectedOrigin) return
  if (typeof event.data !== 'string') return
  const info = readPlayerInfo(event.data)
  if (info?.currentTime !== undefined) emit('time-update', Math.round(info.currentTime * 1000))
  if (info?.playerState !== undefined) emit('play-state', info.playerState === 1)
}

function seek(ms: number): void {
  if (player.value) {
    player.value.currentTime = ms / 1000
    runInBackground(player.value.play(), reportFailure)
    return
  }
  seekFrame(frame.value?.contentWindow, props.embed, ms)
}
</script>

<template>
  <div ref="root" class="embed" @contextmenu.prevent>
    <video
      v-if="props.copy"
      ref="player"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': isHeld }"
      :style="frameStyle"
      :src="props.copy"
      :title="props.words.playing"
      controls
      preload="metadata"
      @contextmenu.prevent
      @timeupdate="onVideoTimeUpdate"
      @seeked="onVideoTimeUpdate"
    ></video>

    <iframe
      v-else-if="props.embed"
      ref="frame"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': isHeld }"
      :style="frameStyle"
      :src="props.embed"
      :title="props.words.playing"
      scrolling="no"
      sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"
      allow="fullscreen; picture-in-picture; encrypted-media"
      @load="onFrameLoad"
    ></iframe>

    <div
      v-if="props.copy || props.embed"
      class="embed__handle"
      role="separator"
      tabindex="0"
      aria-orientation="horizontal"
      :aria-label="props.words.taller"
      :aria-valuemin="LEAST"
      :aria-valuemax="tallest"
      :aria-valuenow="handleValue"
      :data-state="isHeld ? 'drag' : undefined"
      :title="props.words.taller"
      @pointerdown="onPointerDown"
      @keydown="onHandleKeydown"
    ></div>
  </div>
</template>

<style scoped>
/* What is at the address sits at the top of the tab and keeps its height: the
   transcript below it is what scrolls. */
.embed {
  display: flex;
  flex-direction: column;
  flex: none;
  min-inline-size: 0;
  min-block-size: 0;
  /* What is drawn here is as wide as the tab and no wider. Both axes are named:
     one axis hidden makes the other scroll, and a player is not a scroller. */
  overflow: hidden;
}

/* Sixteen by nine until a person draws it otherwise, and then whatever they
   drew. */
.embed__frame {
  flex: none;
  display: block;
  inline-size: 100%;
  max-inline-size: 100%;
  block-size: auto;
  aspect-ratio: 16 / 9;
  max-block-size: min(40vh, 52cqh);
  border: 0;
  background: var(--numen-media-backdrop);
}

/* While the bar is held, the player takes no pointers: a frame that took one
   would take the release with it. */
.embed__frame--held {
  pointer-events: none;
}

.embed__frame--drawn {
  aspect-ratio: auto;
  max-block-size: none;
}

/* The line under the player is what it is drawn taller and shorter by, and it
   is the line between two panels: one stroke of the rule, with the reach a
   pointer is caught by hanging under it. */
.embed__handle {
  --reach: 0.55rem;

  position: relative;
  flex: none;
  block-size: var(--numen-stroke);
  background: var(--numen-rule);
  cursor: ns-resize;
  touch-action: none;
}

.embed__handle::after {
  content: '';
  position: absolute;
  inset-inline: 0;
  inset-block: calc(var(--reach) * -1);
}

.embed__handle[data-state='drag'] {
  background: var(--numen-ring);
}

/* A finger is caught from further out than a pointer. */
@media (pointer: coarse) {
  .embed__handle {
    --reach: 0.95rem;
  }
}
</style>
