<script setup lang="ts">
/**
 * Where the window's players are drawn: over the tab that asked for each, at
 * the box that tab reported.
 *
 * The layer belongs to the window and not to any tab, so a tab moved between
 * panes, put behind another or brought back is the same element throughout.
 */
import { held as players, type Played, type Playing } from './players'

const props = defineProps<{ words: { readonly playing: string } }>()

/** Where one player stands, and out of the way while no tab draws it. */
const at = (one: Playing) =>
  one.box
    ? {
        insetBlockStart: `${one.box.y}px`,
        insetInlineStart: `${one.box.x}px`,
        inlineSize: `${one.box.width}px`,
        blockSize: `${one.box.height}px`,
      }
    : undefined

const drew = (one: Playing, element: Element | null) => {
  players.drew(one.id, (element as Played | null) ?? null)
}
</script>

<template>
  <div class="players">
    <div
      v-for="one in players.all.value"
      :key="one.id"
      class="players__at"
      :hidden="one.box === null"
      :style="at(one)"
      @contextmenu.prevent
    >
      <video
        v-if="one.copy"
        :ref="(element) => drew(one, element as Element | null)"
        class="players__played"
        :src="one.copy"
        :title="props.words.playing"
        controls
        preload="metadata"
      ></video>

      <iframe
        v-else-if="one.embed"
        :ref="(element) => drew(one, element as Element | null)"
        class="players__played"
        :src="one.embed"
        :title="props.words.playing"
        sandbox="allow-scripts allow-same-origin allow-popups"
        allow="fullscreen; picture-in-picture"
      ></iframe>
    </div>
  </div>
</template>

<style scoped>
/* The layer takes no pointers of its own: what is between the players belongs
   to whatever is drawn under it. */
.players {
  position: fixed;
  inset: 0;
  /* Over the panes, which stand at one and come before this, and under the mark
     of where a dragged tab would land, which stands at two. */
  z-index: 1;
  pointer-events: none;
}

.players__at {
  position: absolute;
  overflow: hidden;
  pointer-events: auto;
  background: #000;
}

.players__played {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  border: 0;
}
</style>
