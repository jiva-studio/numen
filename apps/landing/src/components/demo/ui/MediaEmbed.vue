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
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { getDraggedHeight, getFrameOrigin, getTallest, LEAST, parseJson, seekFrame } from '../lib/frame'
import { createHeightStore, type HeightStore } from '../model/heightStore'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    embed: string
    copy: string
    words: { readonly playing: string; readonly taller: string }
    store?: HeightStore
  }>(),
  { store: () => createHeightStore() },
)

const emit = defineEmits<{
  'time-update': [ms: number]
  'play-state': [isPlaying: boolean]
}>()

/* --------------------------------- State ---------------------------------- */
const root = useTemplateRef<HTMLElement>('root')
const frame = useTemplateRef<HTMLIFrameElement>('frame')
const player = useTemplateRef<HTMLVideoElement>('player')

/** How tall the player is drawn, remembered on this machine. */
const tall = ref(props.store.read())
const drawn = ref<{ bar: HTMLElement; pointer: number; from: number; was: number } | null>(null)
const frameStyle = computed(() =>
  tall.value === null ? undefined : { blockSize: `${tall.value}px` },
)
/** The rendered height of the player and the tallest it can be drawn, for the resize handle. */
const renderedHeight = ref<number | null>(null)
const tallest = ref(LEAST)
const handleValue = computed(() =>
  Math.min(tallest.value, Math.max(LEAST, renderedHeight.value ?? tall.value ?? LEAST)),
)

watch(
  () => player.value ?? frame.value,
  (element, _, onCleanup) => {
    if (!element) return
    const sizes = new ResizeObserver(() => {
      renderedHeight.value = Math.round(element.clientHeight)
      tallest.value = getTallest(getPaneHeight())
      if (tall.value !== null) tall.value = Math.min(tallest.value, tall.value)
    })
    sizes.observe(element)
    const pane = findPane()
    if (pane) sizes.observe(pane)
    onCleanup(() => sizes.disconnect())
  },
  { flush: 'post' },
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (tall.value !== null) {
    tall.value = Math.min(getTallest(getPaneHeight()), Math.max(LEAST, tall.value))
  }
  window.addEventListener('message', onWindowMessage)
})

// A drag the page is taken out from under lets go of the bar it was holding.
onBeforeUnmount(() => {
  window.removeEventListener('message', onWindowMessage)
  onPointerUp()
})

/* -------------------------------- Handlers -------------------------------- */
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
  tall.value = getDraggedHeight(held.was, at.clientY - held.from, getPaneHeight())
}

function onPointerUp(): void {
  const held = drawn.value
  if (!held) return
  drawn.value = null
  held.bar.removeEventListener('pointermove', onPointerMove)
  held.bar.removeEventListener('pointerup', onPointerUp)
  held.bar.removeEventListener('pointercancel', onPointerUp)
  if (held.bar.hasPointerCapture(held.pointer)) held.bar.releasePointerCapture(held.pointer)
  if (tall.value !== null) props.store.write(tall.value)
}

function onHandleKeydown(event: KeyboardEvent): void {
  if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return
  event.preventDefault()
  const bar = event.currentTarget as HTMLElement
  const current = tall.value ?? bar.previousElementSibling?.clientHeight ?? LEAST
  const delta = event.key === 'ArrowDown' ? 16 : -16
  tall.value = getDraggedHeight(current, delta, getPaneHeight())
  props.store.write(tall.value)
}

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
  const payload = parseJson(event.data) as {
    event?: string
    info?: { currentTime?: number; playerState?: number }
  } | null | undefined
  if (payload?.event === 'infoDelivery' && payload.info) {
    if (typeof payload.info.currentTime === 'number') {
      emit('time-update', Math.round(payload.info.currentTime * 1000))
    }
    if (typeof payload.info.playerState === 'number') {
      emit('play-state', payload.info.playerState === 1)
    }
  }
}

/* -------------------------------- Helpers --------------------------------- */
function findPane(): HTMLElement | null {
  for (let at = root.value?.parentElement ?? null; at; at = at.parentElement) {
    if (getComputedStyle(at).containerType.split(' ').includes('size')) return at
  }
  return null
}

function getPaneHeight(): number {
  return findPane()?.clientHeight || window.innerHeight
}
function seek(ms: number): void {
  if (player.value) {
    player.value.currentTime = ms / 1000
    player.value.play().catch((error: unknown) => {
      console.error(error)
    })
    return
  }
  seekFrame(frame.value?.contentWindow, props.embed, ms)
}

defineExpose({ seek })
</script>

<template>
  <div ref="root" class="embed" @contextmenu.prevent>
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
      @seeked="onVideoTimeUpdate"
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
      :data-state="drawn ? 'drag' : undefined"
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
