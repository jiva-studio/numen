<script setup lang="ts">
/**
 * Displays the graph neighborhood of notes and handles graph interaction gestures.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Menu, optionsForType, Plex, useTypeSize, Waiting } from '@numen/ui'
import type { MenuTrigger, PlexRelation, PlexDestination } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'
import PlexQuickLinkPopover from './ui/PlexQuickLinkPopover.vue'
import PlexLinkInspectorPopover from './ui/PlexLinkInspectorPopover.vue'
import { ITEMS, NONE } from './lib/menu'
import { findIcon, getNoteIcon } from './lib/icons'
import type { DirectionalLinkItem, PlexTabState } from './types'
import { WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
defineOptions({ name: 'PlexTab' })
const props = defineProps<{ state: PlexTabState }>()

/* --------------------------------- State ---------------------------------- */
// The tab's state outlives this component, so what it holds is bound once here
// and the template unwraps it.
const {
  dragged,
  empty,
  menu,
  quickLink,
  linkInspector,
  maxHeadings,
  picture: neighbourhood,
} = props.state

/**
 * How large the picture is drawn, and how many headings a node hangs at once. A
 * node's label is set in the window's type, so the boxes and the clearances
 * between them are handed in to hold it.
 */
const type = useTypeSize()
const options = computed(() => ({
  ...optionsForType(type.value),
  maxHeadings: maxHeadings.value,
  // Cast icon function to match PlexOptions if needed
  icon: getNodeIcon as unknown as (node: string) => LucideIcon | null,
}))

/** What the menu offers: on a node, or off every node. */
const items = computed(() => (menu.value?.node === null ? NONE : ITEMS))

const lastPointer = ref<{ x: number; y: number }>({ x: 300, y: 300 })

const rootElement = useTemplateRef<HTMLElement>('rootElement')
const picture = useTemplateRef<{ focusNode: (id: string) => void }>('picture')

/** The node a quick link leaves and the point it is drawn to. */
const pending = computed(() =>
  quickLink.value ? { from: quickLink.value.from, at: quickLink.value.at } : null,
)

/* -------------------------------- Handlers -------------------------------- */
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

function onPointerUp(event: PointerEvent) {
  const rect = rootElement.value?.getBoundingClientRect()
  if (rect) {
    lastPointer.value = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top,
    }
  } else {
    lastPointer.value = { x: event.clientX, y: event.clientY }
  }
}

function onCreateNode(from: string, relation: PlexRelation, at?: { x: number; y: number }) {
  void props.state.createNode(from, relation, at ?? lastPointer.value)
}

function onLinkNodes(from: string, to: string, relation: PlexRelation) {
  void props.state.joinNodes(from, to, relation)
}

function onBringNodes(nodes: readonly string[], relation: PlexRelation) {
  void props.state.dropNodes(nodes, relation)
}

function onOpenMenu(node: string, at: { x: number; y: number }, opening: MenuTrigger) {
  props.state.openMenu({ node, at, opening })
}

function onShowNode(node: string, how: PlexDestination) {
  props.state.openNode(node, how)
}

function onEnterHeading(node: string, heading: string) {
  props.state.openHeading(node, heading)
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

function onSelectQuickLinkNote(path: string) {
  void props.state.confirmQuickLink(path, true)
}

function onCreateQuickLinkNote(title: string) {
  void props.state.confirmQuickLink(title, false)
}

function onDismissQuickLink() {
  props.state.dismissQuickLink()
}

function onEdgeMenu(pair: string, at: { x: number; y: number }) {
  const rect = rootElement.value?.getBoundingClientRect()
  if (rect) {
    const localX = at.x - rect.left
    const localY = at.y - rect.top
    props.state.openLinkInspector(pair, { x: localX, y: localY })
  } else {
    props.state.openLinkInspector(pair, at)
  }
}

function onSaveLinkInspector(rows: readonly DirectionalLinkItem[]) {
  void props.state.saveLinkInspector(rows)
}

function onDismissLinkInspector() {
  props.state.dismissLinkInspector()
}

/* -------------------------------- Helpers --------------------------------- */
/**
 * What a node is drawn before its title, and nothing for an ordinary note. A
 * deck, a stencil and a preset carry the icon the tree draws them under.
 */
function getNodeIcon(node: string): LucideIcon | null {
  const type = props.state.getType(node)
  return type === 'note' ? null : getNoteIcon(type)
}

/** A menu put away, and the keyboard back on the node it was asked from. */
function closeMenu(id?: string) {
  const node = menu.value?.node ?? null
  if (id === undefined) props.state.dismiss()
  else props.state.chooseMenuItem(id)
  if (node !== null) picture.value?.focusNode(node)
}
</script>

<template>
  <div ref="rootElement" class="plex" @contextmenu="onContextMenu" @pointerup.capture="onPointerUp">
    <p v-if="props.state.view.error.value" class="caution">
      {{ props.state.view.error.value }}
    </p>

    <!-- Until the first neighbourhood lands, an empty plex and one nobody has
         answered for yet are drawn the same way. -->
    <Waiting v-if="props.state.view.isLoading.value" :label="words.loading" />

    <Plex
      v-else-if="neighbourhood"
      ref="picture"
      class="plex__picture"
      :neighbourhood="neighbourhood!"
      :options="options"
      :creatable="props.state.creatable"
      :pending="pending"
      :dragged="dragged"
      :drop-name="words.dropName"
      :headings="props.state.getHeadings"
      @activate="onActivateNode"
      @create="onCreateNode"
      @link="onLinkNodes"
      @bring="onBringNodes"
      @menu="onOpenMenu"
      @show="onShowNode"
      @enter="onEnterHeading"
      @edge-menu="onEdgeMenu"
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
      :trigger="menu!.opening"
      open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component :is="findIcon(id)" v-if="findIcon(id)" class="plex__icon" aria-hidden="true" />
      </template>
    </Menu>

    <PlexQuickLinkPopover
      v-if="quickLink"
      :at="quickLink.at"
      :relation="quickLink.relation"
      :search="props.state.searchNotes"
      @select-note="onSelectQuickLinkNote"
      @create-note="onCreateQuickLinkNote"
      @dismiss="onDismissQuickLink"
    />

    <PlexLinkInspectorPopover
      v-if="linkInspector"
      :request="linkInspector"
      @save="onSaveLinkInspector"
      @dismiss="onDismissLinkInspector"
    />
  </div>
</template>

<style scoped>
/* The picture takes what the band above it leaves. */
.plex {
  position: relative;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  overflow: hidden;
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
