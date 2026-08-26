<script setup lang="ts">
/**
 * A plex tab: the picture, the menu on a node of it, and what this tab could
 * not show.
 *
 * Every gesture is handed to what the tab holds. The menu stands on a node of
 * this picture and goes when the picture does. A note this tab cannot draw is
 * this tab's own trouble, and is said in it.
 */
import { computed } from 'vue'
import { Menu, optionsForType, Plex, useTypeSize } from '@numen/ui'
import type { MenuOpening, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import { ITEMS } from './menu'
import type { Held } from './kind'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

/**
 * How large the picture is drawn. A node's label is set in the window's type,
 * so the boxes and the clearances between them are handed in to hold it.
 */
const type = useTypeSize()
const options = computed(() => optionsForType(type.value))
</script>

<template>
  <div class="plex">
    <p v-if="props.held.view.trouble.value" class="warning">
      {{ props.held.view.trouble.value }}
    </p>

    <Plex
      v-if="props.held.picture.value"
      class="plex__picture"
      :neighbourhood="props.held.picture.value!"
      :options="options"
      :creatable="props.held.creatable"
      :carried="props.held.carried.value"
      :carried-name="words.carried"
      @activate="(node: string) => props.held.activate(node)"
      @create="(from: string, seat: PlexRelatedSeat) => void props.held.made(from, seat)"
      @link="
        (from: string, to: string, seat: PlexRelatedSeat) => void props.held.joined(from, to, seat)
      "
      @bring="(carried: string, seat: PlexRelatedSeat) => void props.held.brought(carried, seat)"
      @menu="
        (
          node: string,
          at: { x: number; y: number },
          from: SVGGElement,
          opening: MenuOpening,
        ) => props.held.asks({ node, at, from, opening })
      "
      @show="(node: string, how: PlexShowing) => props.held.opens(node, how)"
      @dismiss="props.held.dismiss()"
    />

    <Menu
      v-if="props.held.menu.value"
      :items="ITEMS"
      :at="props.held.menu.value!.at"
      :from="props.held.menu.value!.from"
      :opening="props.held.menu.value!.opening"
      open
      @choose="(id: string) => props.held.chose(id)"
      @dismiss="props.held.dismiss()"
    />
  </div>
</template>

<style scoped>
/* The picture takes what the band above it leaves. */
.plex {
  display: flex;
  flex-direction: column;
  block-size: 100%;
}

.plex__picture {
  flex: 1;
  min-block-size: 0;
}

/* A warning carries a filesystem path, and a long one breaks where it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: 0.8rem;
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}
</style>
