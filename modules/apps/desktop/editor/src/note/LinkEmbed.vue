<script setup lang="ts">
/**
 * What a link note points at, played at the top of its tab: a copy on this disk
 * where there is one, a frame where there is not, and the address itself where
 * nothing plays at all.
 *
 * The transcript stands under this and is where a moment is chosen. What that
 * moment does is here: a copy is seeked, and a frame is told through the
 * messages a player takes from the page holding it, so nothing that host serves
 * runs in this window.
 */
import { onBeforeUnmount, ref, useTemplateRef } from 'vue'
import type { Address } from '../core'

const props = defineProps<{
  address: Address
  /**
   * Where a copy of it on this disk is played from, and nothing where there is
   * none. A copy is played in place of the frame: it plays offline, and nothing
   * of the site it came from is loaded to play it.
   */
  copy: string
  /** What the frame and the bar that resizes it are called. */
  words: { readonly playing: string; readonly taller: string }
}>()

const frame = useTemplateRef<HTMLIFrameElement>('frame')
const player = useTemplateRef<HTMLVideoElement>('player')

/** How tall the player is drawn, remembered on this machine. */
const HEIGHT = 'numen.embed.height'
const LEAST = 120

/**
 * How tall a person drew the player last, and nothing where they have not drawn
 * it: sixteen by nine is what it is until somebody says otherwise.
 */
const read = (): number | null => {
  try {
    const kept = Number(localStorage.getItem(HEIGHT))
    return kept >= LEAST ? kept : null
  } catch {
    return null
  }
}

const tall = ref(read())
const drawn = ref<{ bar: HTMLElement; pointer: number; from: number; was: number } | null>(null)

/**
 * The bar under the player is taken hold of, and the player follows it.
 *
 * The pointer is captured by the bar and the player stops taking pointers at
 * all: a frame is a document of its own, and one dragged across takes every
 * move and the release with it — so the bar would follow the pointer only while
 * it stayed off the player, and would never hear that it was let go.
 */
const draws = (at: PointerEvent): void => {
  const bar = at.currentTarget as HTMLElement
  const held = bar.previousElementSibling
  bar.setPointerCapture(at.pointerId)
  drawn.value = {
    bar,
    pointer: at.pointerId,
    from: at.clientY,
    was: held?.clientHeight ?? LEAST,
  }
  bar.addEventListener('pointermove', drawing)
  bar.addEventListener('pointerup', drew)
  bar.addEventListener('pointercancel', drew)
}

const drawing = (at: PointerEvent): void => {
  const held = drawn.value
  if (!held) return
  const most = Math.max(LEAST, window.innerHeight - 160)
  tall.value = Math.min(most, Math.max(LEAST, held.was + at.clientY - held.from))
}

const drew = (): void => {
  const held = drawn.value
  if (!held) return
  drawn.value = null
  held.bar.removeEventListener('pointermove', drawing)
  held.bar.removeEventListener('pointerup', drew)
  held.bar.removeEventListener('pointercancel', drew)
  if (held.bar.hasPointerCapture(held.pointer)) held.bar.releasePointerCapture(held.pointer)
  try {
    if (tall.value !== null) localStorage.setItem(HEIGHT, String(tall.value))
  } catch {
    // A machine that keeps nothing for this page draws the player this tall
    // until it is closed, which is the whole of what is lost.
  }
}

onBeforeUnmount(drew)

/** Played from a moment, in milliseconds: the copy where there is one. */
const seeks = (ms: number): void => {
  if (player.value) {
    player.value.currentTime = ms / 1000
    void player.value.play()
    return
  }
  frame.value?.contentWindow?.postMessage(
    JSON.stringify({ event: 'command', func: 'seekTo', args: [ms / 1000, true] }),
    new URL(props.address.embed).origin,
  )
}

defineExpose({ seeks })
</script>

<template>
  <div class="embed" @contextmenu.prevent>
    <video
      v-if="props.copy"
      ref="player"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': drawn !== null }"
      :style="tall === null ? undefined : { blockSize: `${tall}px` }"
      :src="props.copy"
      :title="props.words.playing"
      controls
      preload="metadata"
    ></video>

    <iframe
      v-else-if="props.address.embed"
      ref="frame"
      class="embed__frame"
      :class="{ 'embed__frame--drawn': tall !== null, 'embed__frame--held': drawn !== null }"
      :style="tall === null ? undefined : { blockSize: `${tall}px` }"
      :src="props.address.embed"
      :title="props.words.playing"
      sandbox="allow-scripts allow-same-origin allow-popups"
      allow="fullscreen; picture-in-picture"
    ></iframe>

    <p v-else class="embed__address">{{ props.address.url }}</p>

    <div
      v-if="props.copy || props.address.embed"
      class="embed__handle"
      role="separator"
      aria-orientation="horizontal"
      :title="props.words.taller"
      @pointerdown="draws"
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
  /* What is drawn here is as wide as the tab and no wider: a player is given
     the width it has, and the tab scrolls down its text and never across. */
  overflow-x: hidden;
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
  background: #000;
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

/* The rule under the player is what it is drawn taller and shorter by: one
   line, with room around it for a pointer to take hold of. */
.embed__handle {
  flex: none;
  block-size: 7px;
  cursor: row-resize;
  background: linear-gradient(var(--numen-rule), var(--numen-rule)) center / 100% 1px no-repeat;
  touch-action: none;
}

.embed__handle:hover {
  background: linear-gradient(var(--numen-hushed), var(--numen-hushed)) center / 100% 1px no-repeat;
}

.embed__address {
  margin: 0;
  padding: var(--numen-inset) var(--numen-gutter);
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  overflow-wrap: anywhere;
}
</style>
