<script setup lang="ts">
/**
 * A plex tab: the picture, the menu on a node of it, and what this tab could
 * not show.
 *
 * Every gesture is handed to what the tab holds. The menu stands on a node of
 * this picture and goes when the picture does. A note this tab cannot draw is
 * this tab's own trouble, and is said in it.
 */
import { computed, useTemplateRef } from 'vue'
import { Menu, optionsForType, Plex, useTypeSize } from '@numen/ui'
import type { MenuOpening, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'
import { ITEMS, NONE } from './menu'
import Caution from '../Caution.vue'
import { iconFor, iconOfNote } from '../icons'
import type { Held } from './kind'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

/**
 * How large the picture is drawn, and how many parts a node hangs at once. A
 * node's label is set in the window's type, so the boxes and the clearances
 * between them are handed in to hold it.
 */
const type = useTypeSize()
const options = computed(() => ({
  ...optionsForType(type.value),
  maxParts: props.held.mostParts(),
}))

/**
 * What a node is drawn before its title, and nothing for an ordinary note. A
 * deck, a stencil and a preset carry the icon the tree draws them under.
 */
const nodeIcon = (node: string): LucideIcon | null => {
  const type = props.held.typeOf(node)
  return type === 'note' ? null : iconOfNote(type)
}

/** What the menu offers: on a node, or off every node. */
const items = computed(() => (props.held.menu.value?.node === null ? NONE : ITEMS))

/**
 * A menu asked for over a tab drawing no picture, which is a vault holding no
 * note to draw one around. A tab drawing one leaves the picture to answer, and
 * so does a vault that has notes and has not been read yet.
 */
const asks = (event: MouseEvent) => {
  if (!props.held.empty.value) return
  event.preventDefault()
  props.held.asks({
    node: null,
    at: { x: event.clientX, y: event.clientY },
    opening: 'pointer',
  })
}

const picture = useTemplateRef<{ focusNode: (id: string) => void }>('picture')

/** A menu put away, and the keyboard back on the node it was asked from. */
const closed = (chose?: string) => {
  const node = props.held.menu.value?.node ?? null
  if (chose === undefined) props.held.dismiss()
  else props.held.chose(chose)
  if (node !== null) picture.value?.focusNode(node)
}
</script>

<template>
  <div class="plex" @contextmenu="asks">
    <Caution v-if="props.held.view.trouble.value">
      {{ props.held.view.trouble.value }}
    </Caution>

    <Plex
      v-if="props.held.picture.value"
      ref="picture"
      class="plex__picture"
      :neighbourhood="props.held.picture.value!"
      :options="options"
      :creatable="props.held.creatable"
      :carried="props.held.carried.value"
      :carried-name="words.carried"
      :parts="props.held.partsOf"
      @activate="(node: string) => props.held.activate(node)"
      @create="(from: string, seat: PlexRelatedSeat) => void props.held.made(from, seat)"
      @link="
        (from: string, to: string, seat: PlexRelatedSeat) => void props.held.joined(from, to, seat)
      "
      @bring="
        (carried: readonly string[], seat: PlexRelatedSeat) =>
          void props.held.brought(carried, seat)
      "
      @menu="
        (node: string, at: { x: number; y: number }, opening: MenuOpening) =>
          props.held.asks({ node, at, opening })
      "
      @show="(node: string, how: PlexShowing) => props.held.opens(node, how)"
      @enter="(node: string, part: string) => props.held.entered(node, part)"
      @dismiss="props.held.dismiss()"
    >
      <!-- A deck, a stencil and a preset are drawn as the tree draws them. An
           ordinary note is drawn its title and nothing before it. -->
      <template #icon="{ node }: { node: { id: string } }">
        <component
          :is="nodeIcon(node.id)"
          v-if="nodeIcon(node.id)"
          class="plex__icon"
          aria-hidden="true"
        />
      </template>
    </Plex>

    <Menu
      v-if="props.held.menu.value"
      :items="items"
      :at="props.held.menu.value!.at"
      :opening="props.held.menu.value!.opening"
      open
      @choose="(id: string) => closed(id)"
      @dismiss="closed()"
    >
      <template #icon="{ id }">
        <component :is="iconFor(id)" v-if="iconFor(id)" class="plex__icon" aria-hidden="true" />
      </template>
    </Menu>
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

/* Lucide draws on a 24 grid, and the stroke is given in those units.
   A node's title stands in a `foreignObject`, and nothing drawn in one is given
   an opacity of its own: WebKit paints what it makes a layer of at the corner
   of the picture. Anything quieter is asked for in the colour. */
.plex__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
