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
import { computed, onBeforeUnmount, ref, useTemplateRef } from 'vue'

// --- Props & Emits ---
const props = defineProps<{
  /** Where a frame plays what is at the address, and nothing where a copy is. */
  embed: string
  /**
   * Where a copy of it on this disk is played from, and nothing where there is
   * none. A copy is played in place of the frame: it plays offline, and nothing
   * of the site it came from is loaded to play it.
   */
  copy: string
  /** What the frame and the bar that resizes it are called. */
  words: { readonly playing: string; readonly taller: string }
}>()

const emit = defineEmits<{
  /** Where the copy stands, in milliseconds, as it plays. A frame says nothing. */
  'time-update': [ms: number]
}>()

// --- State ---
const frame = useTemplateRef<HTMLIFrameElement>('frame')
const player = useTemplateRef<HTMLVideoElement>('player')

/** How tall the player is drawn, remembered on this machine. */
const HEIGHT = 'numen.embed.height'
const LEAST = 120

const tall = ref(readHeight())
const drawn = ref<{ bar: HTMLElement; pointer: number; from: number; was: number } | null>(null)
const frameStyle = computed(() =>
  tall.value === null ? undefined : { blockSize: `${tall.value}px` },
)

// --- Handlers ---
/**
 * The bar under the player is taken hold of, and the player follows it.
 *
 * The pointer is captured by the bar and the player stops taking pointers at
 * all: a frame is a document of its own, and one dragged across takes every
 * move and the release with it — so the bar would follow the pointer only while
 * it stayed off the player, and would never hear that it was let go.
 */
function onPointerDown(at: PointerEvent): void {
  const bar = at.currentTarget as HTMLElement
  const held = bar.previousElementSibling
  bar.setPointerCapture(at.pointerId)
  drawn.value = {
    bar,
    pointer: at.pointerId,
    from: at.clientY,
    was: held?.clientHeight ?? LEAST,
  }
  bar.addEventListener('pointermove', onPointerMove)
  bar.addEventListener('pointerup', onPointerUp)
  bar.addEventListener('pointercancel', onPointerUp)
}

function onPointerMove(at: PointerEvent): void {
  const held = drawn.value
  if (!held) return
  const most = Math.max(LEAST, window.innerHeight - 160)
  tall.value = Math.min(most, Math.max(LEAST, held.was + at.clientY - held.from))
}

function onPointerUp(): void {
  const held = drawn.value
  if (!held) return
  drawn.value = null
  held.bar.removeEventListener('pointermove', onPointerMove)
  held.bar.removeEventListener('pointerup', onPointerUp)
  held.bar.removeEventListener('pointercancel', onPointerUp)
  if (held.bar.hasPointerCapture(held.pointer)) held.bar.releasePointerCapture(held.pointer)
  try {
    if (tall.value !== null) localStorage.setItem(HEIGHT, String(tall.value))
  } catch {
    // A machine that keeps nothing for this page draws the player this tall
    // until it is closed, which is the whole of what is lost.
  }
}

function onVideoTimeUpdate(event: Event): void {
  const video = event.target as HTMLVideoElement
  emit('time-update', Math.round(video.currentTime * 1000))
}

onBeforeUnmount(onPointerUp)

// --- Helpers ---
/**
 * How tall a person drew the player last, and nothing where they have not drawn
 * it: sixteen by nine is what it is until somebody says otherwise.
 */
function readHeight(): number | null {
  try {
    const kept = Number(localStorage.getItem(HEIGHT))
    return kept >= LEAST ? kept : null
  } catch {
    // A browser that keeps nothing for this page has no height to give back.
    return null
  }
}

/** Played from a moment, in milliseconds: the copy where there is one. */
function seek(ms: number): void {
  if (player.value) {
    player.value.currentTime = ms / 1000
    void player.value.play()
    return
  }
  frame.value?.contentWindow?.postMessage(
    JSON.stringify({ event: 'command', func: 'seekTo', args: [ms / 1000, true] }),
    new URL(props.embed).origin,
  )
}

defineExpose({ seek })
</script>

<template>
  <div class="embed" @contextmenu.prevent>
    <video
      v-if="props.copy"
      ref="player"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': drawn !== null }"
      :style="frameStyle"
      :src="props.copy"
      :title="props.words.playing"
      controls
      preload="metadata"
      @contextmenu.prevent
      @timeupdate="onVideoTimeUpdate"
    ></video>

    <iframe
      v-else-if="props.embed"
      ref="frame"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': drawn !== null }"
      :style="frameStyle"
      :src="props.embed"
      :title="props.words.playing"
      scrolling="no"
      sandbox="allow-scripts allow-same-origin allow-popups"
      allow="fullscreen; picture-in-picture"
    ></iframe>

    <div
      v-if="props.copy || props.embed"
      class="embed__handle"
      role="separator"
      aria-orientation="horizontal"
      :data-state="drawn ? 'drag' : undefined"
      :title="props.words.taller"
      @pointerdown="onPointerDown"
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
  max-block-size: 40vh;
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
  --reach: 9px;

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
    --reach: 15px;
  }
}
</style>
