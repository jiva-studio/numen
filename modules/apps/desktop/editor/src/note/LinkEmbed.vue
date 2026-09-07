<script setup lang="ts">
/**
 * Where a link note's player stands at the top of its tab: the box it is drawn
 * in, and the rule under it that draws it taller and shorter.
 *
 * The player itself is the window's, drawn in a layer of its own over the box
 * reported here. An address nothing plays is the address itself.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import type { Address } from '../core'
import { held as players } from './players'

const props = defineProps<{
  /** The tab this player belongs to, which it keeps wherever the tab goes. */
  id: string
  address: Address
  /**
   * Where a copy of it on this disk is played from, and nothing where there is
   * none. A copy is played in place of the frame: it plays offline, and nothing
   * of the site it came from is loaded to play it.
   */
  copy: string
  /** What the player and the bar that resizes it are called. */
  words: { readonly playing: string; readonly taller: string }
}>()

const box = useTemplateRef<HTMLElement>('box')

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

/** Whether anything plays here at all, or the address stands on its own. */
const plays = computed(() => Boolean(props.copy || props.address.embed))

/**
 * The box is reported to the layer as it is on the screen. A box of no size is
 * a tab standing behind another, and its player is drawn nowhere and kept.
 */
const measures = (): void => {
  const held = box.value
  if (!held || !plays.value) {
    players.hides(props.id)
    return
  }
  const at = held.getBoundingClientRect()
  players.draws({
    id: props.id,
    embed: props.copy ? '' : props.address.embed,
    copy: props.copy,
    title: props.words.playing,
    box:
      at.width === 0 || at.height === 0
        ? null
        : { x: at.x, y: at.y, width: at.width, height: at.height },
  })
}

let watching: ResizeObserver | null = null

onMounted(() => {
  measures()
  if (typeof ResizeObserver === 'function' && box.value) {
    watching = new ResizeObserver(measures)
    watching.observe(box.value)
  }
  window.addEventListener('resize', measures)
  // A pane scrolled under the box moves it without changing its size, and the
  // scroll is heard wherever it happens.
  window.addEventListener('scroll', measures, true)
})

onBeforeUnmount(() => {
  drew()
  watching?.disconnect()
  watching = null
  window.removeEventListener('resize', measures)
  window.removeEventListener('scroll', measures, true)
  players.hides(props.id)
})

// The box is measured once it is drawn, so what is reported is where it stands.
watch(() => [props.copy, props.address.embed, plays.value] as const, measures, { flush: 'post' })

/**
 * The bar under the player is taken hold of, and the player follows it.
 *
 * The pointer is captured by the bar: a player is a document of its own, and
 * one dragged across takes every move and the release with it.
 */
const draws = (at: PointerEvent): void => {
  const bar = at.currentTarget as HTMLElement
  bar.setPointerCapture(at.pointerId)
  drawn.value = {
    bar,
    pointer: at.pointerId,
    from: at.clientY,
    was: box.value?.clientHeight ?? LEAST,
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

/** Played from a moment, in milliseconds. */
const seeks = (ms: number): void => players.seeks(props.id, ms)

defineExpose({ seeks })
</script>

<template>
  <div class="embed">
    <div
      v-if="plays"
      ref="box"
      class="embed__box"
      :class="{ 'embed__box--drawn': tall !== null }"
      :style="tall === null ? undefined : { blockSize: `${tall}px` }"
    ></div>

    <p v-else class="embed__address">{{ props.address.url }}</p>

    <!-- The strip under the player: the rule it is drawn taller and shorter by,
         and what can be asked over the address standing at its end. -->
    <div class="embed__strip">
      <div
        v-if="plays"
        class="embed__handle"
        role="separator"
        aria-orientation="horizontal"
        :title="props.words.taller"
        @pointerdown="draws"
      ></div>

      <slot name="end" />
    </div>
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
   drew. The player is drawn over this. */
.embed__box {
  flex: none;
  inline-size: 100%;
  max-inline-size: 100%;
  block-size: auto;
  aspect-ratio: 16 / 9;
  max-block-size: 40vh;
  background: #000;
}

.embed__box--drawn {
  aspect-ratio: auto;
  max-block-size: none;
}

/* One row under the player, as tall as what stands at its end. */
.embed__strip {
  display: flex;
  flex: none;
  align-items: stretch;
  min-block-size: 7px;
}

/* The rule under the player is what it is drawn taller and shorter by: one
   line, with room around it for a pointer to take hold of. */
.embed__handle {
  flex: 1;
  min-inline-size: 0;
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
