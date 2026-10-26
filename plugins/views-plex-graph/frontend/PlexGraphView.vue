<script setup lang="ts">
/**
 * Displays the graph neighborhood of notes and handles graph interaction gestures.
 */
import { computed } from 'vue'
import { Menu, Waiting } from '@numen/ui'
import { optionsForType, Plex, useTypeSize } from './plex'
import type { MenuTrigger } from '@numen/ui'
import type { PlexRelation, PlexDestination } from './plex'
import PlexQuickLinkPopover from './ui/PlexQuickLinkPopover.vue'
import PlexLinkInspectorPopover from './ui/PlexLinkInspectorPopover.vue'
import type { DirectionalLinkItem, PlexTabState } from './types'
import { WORDS as words } from './words'
import { usePlexCanvas } from './model/usePlexCanvas'

/* ----------------------------- Props & Emits ------------------------------ */
defineOptions({ name: 'PlexTab' })
const props = defineProps<{ state: PlexTabState }>()

/* --------------------------------- State ---------------------------------- */
const { dragged, menu, quickLink, linkInspector, maxHeadings, picture: neighbourhood } = props.state

const type = useTypeSize()
const options = computed(() => ({
  ...optionsForType(type.value),
  maxHeadings: maxHeadings.value,
}))

const { menuItems: items, getMenuIcon, getNodeIcon } = props.state

const { rootElement, picture, lastPointer, onPointerUp, onContextMenu, onEdgeMenu, closeMenu } =
  usePlexCanvas({ state: props.state })

const pending = computed(() =>
  quickLink.value ? { from: quickLink.value.from, at: quickLink.value.at } : null,
)

/* -------------------------------- Handlers -------------------------------- */
function onActivateNode(node: string) {
  props.state.activate(node)
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

function onOpenMenu(node: string, position: { x: number; y: number }, opening: MenuTrigger) {
  props.state.openMenu({ node, at: position, opening })
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

function onSaveLinkInspector(rows: readonly DirectionalLinkItem[]) {
  void props.state.saveLinkInspector(rows)
}

function onDismissLinkInspector() {
  props.state.dismissLinkInspector()
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
      <!-- A note of a type a plugin declares is drawn as the tree draws it. An
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
      is-open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component
          :is="getMenuIcon(id)"
          v-if="getMenuIcon(id)"
          class="plex__icon"
          aria-hidden="true"
        />
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

.plex__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
