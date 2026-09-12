<script setup lang="ts">
/**
 * Displays the graph neighborhood of notes and handles graph interaction gestures.
 */
import { computed, useTemplateRef } from 'vue'
import { Menu, optionsForType, Plex, useTypeSize } from '@numen/ui'
import type { MenuOpening, PlexRelatedSeat, PlexShowing } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'
import { ITEMS, NONE } from '../menu'
import { iconFor } from '@/shared/icons'
import { iconOfNote } from '@/entities/note'
import type { PlexTabState } from '../types'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: PlexTabState }>()

// --- State ---
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const { dragged, empty, menu, mostParts, picture: neighbourhood } = props.state

/**
 * How large the picture is drawn, and how many parts a node hangs at once. A
 * node's label is set in the window's type, so the boxes and the clearances
 * between them are handed in to hold it.
 */
const type = useTypeSize()
const options = computed(() => ({
  ...optionsForType(type.value),
  maxParts: mostParts.value,
}))

/** What the menu offers: on a node, or off every node. */
const items = computed(() => (menu.value?.node === null ? NONE : ITEMS))

const picture = useTemplateRef<{ focusNode: (id: string) => void }>('picture')

// --- Handlers ---
/**
 * A menu asked for over a tab drawing no picture, which is a vault holding no
 * note to draw one around. A tab drawing one leaves the picture to answer, and
 * so does a vault that has notes and has not been read yet.
 */
function onContextMenu(event: MouseEvent) {
  if (!empty.value) return
  event.preventDefault()
  props.state.openMenu({
    node: null,
    at: { x: event.clientX, y: event.clientY },
    opening: 'pointer',
  })
}

function onActivateNode(node: string) {
  props.state.activate(node)
}

function onCreateNode(from: string, seat: PlexRelatedSeat) {
  void props.state.createNode(from, seat)
}

function onLinkNodes(from: string, to: string, seat: PlexRelatedSeat) {
  void props.state.joinNodes(from, to, seat)
}

function onBringNodes(dragged: readonly string[], seat: PlexRelatedSeat) {
  void props.state.bringNodes(dragged, seat)
}

function onOpenMenu(node: string, at: { x: number; y: number }, opening: MenuOpening) {
  props.state.openMenu({ node, at, opening })
}

function onShowNode(node: string, how: PlexShowing) {
  props.state.openNode(node, how)
}

function onEnterPart(node: string, part: string) {
  props.state.openPart(node, part)
}

function onDismissPicture() {
  props.state.dismiss()
}

function onChooseMenuItem(id: string) {
  closeMenu(id)
}

function onDismissMenu() {
  closeMenu()
}

// --- Helpers ---
/**
 * What a node is drawn before its title, and nothing for an ordinary note. A
 * deck, a stencil and a preset carry the icon the tree draws them under.
 */
function getNodeIcon(node: string): LucideIcon | null {
  const type = props.state.typeOf(node)
  return type === 'note' ? null : iconOfNote(type)
}

/** A menu put away, and the keyboard back on the node it was asked from. */
function closeMenu(chose?: string) {
  const node = menu.value?.node ?? null
  if (chose === undefined) props.state.dismiss()
  else props.state.chooseMenuItem(chose)
  if (node !== null) picture.value?.focusNode(node)
}
</script>

<template>
  <div class="plex" @contextmenu="onContextMenu">
    <p v-if="props.state.view.error.value" class="caution">
      {{ props.state.view.error.value }}
    </p>

    <Plex
      v-if="neighbourhood"
      ref="picture"
      class="plex__picture"
      :neighbourhood="neighbourhood!"
      :options="options"
      :creatable="props.state.creatable"
      :dragged="dragged"
      :drop-name="words.dropName"
      :parts="props.state.partsOf"
      @activate="onActivateNode"
      @create="onCreateNode"
      @link="onLinkNodes"
      @bring="onBringNodes"
      @menu="onOpenMenu"
      @show="onShowNode"
      @enter="onEnterPart"
      @dismiss="onDismissPicture"
    >
      <!-- A deck, a stencil and a preset are drawn as the tree draws them. An
           ordinary note is drawn its title and nothing before it. -->
      <template #icon="{ node }: { node: { id: string } }">
        <component
          :is="getNodeIcon(node.id)"
          v-if="getNodeIcon(node.id)"
          class="plex__icon"
          aria-hidden="true"
        />
      </template>
    </Plex>

    <Menu
      v-if="menu"
      :items="items"
      :at="menu!.at"
      :opening="menu!.opening"
      open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
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
