<script setup lang="ts">
/**
 * Displays the graph neighborhood of notes and handles graph interaction gestures.
 */
import { computed } from 'vue'
import { Menu, LoadingIndicator } from '@numen/ui'
import { optionsForType, Plex, useTypeSize } from './plex'
import type { MenuTrigger } from '@numen/ui'
import type { PlexRelation, PlexDestination } from './plex'
import PlexQuickLinkPopover from './ui/PlexQuickLinkPopover.vue'
import PlexLinkInspectorPopover from './ui/PlexLinkInspectorPopover.vue'
import type { Connection } from '@numen/plugin-sdk'
import type { PlexTabState } from './types'
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
  props.state.createNode(from, relation, at ?? lastPointer.value).catch(props.state.reportFailure)
}

function onLinkNodes(from: string, to: string, relation: PlexRelation) {
  props.state.joinNodes(from, to, relation).catch(props.state.reportFailure)
}

function onBringNodes(nodes: readonly string[], relation: PlexRelation) {
  props.state.dropNodes(nodes, relation).catch(props.state.reportFailure)
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

function onDismissImage() {
  props.state.dismiss()
}

function onChooseMenuItem(id: string) {
  closeMenu(id)
}

function onDismissMenu() {
  closeMenu()
}

function onSelectQuickLinkNote(path: string) {
  props.state.confirmQuickLink(path, true).catch(props.state.reportFailure)
}

function onCreateQuickLinkNote(title: string) {
  props.state.confirmQuickLink(title, false).catch(props.state.reportFailure)
}

function onDismissQuickLink() {
  props.state.dismissQuickLink()
}

function onSaveLinkInspector(connection: Connection) {
  props.state.saveLinkInspector(connection).catch(props.state.reportFailure)
}

function onRemoveLinkInspector() {
  props.state.removeLinkInspector().catch(props.state.reportFailure)
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
    <LoadingIndicator v-if="props.state.view.isLoading.value" :label="words.loading" />

    <Plex
      v-else-if="neighbourhood"
      ref="picture"
      class="plex__image"
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
      @dismiss="onDismissImage"
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
      @remove="onRemoveLinkInspector"
      @dismiss="onDismissLinkInspector"
    />
  </div>
</template>

<style scoped>
/* The graph takes the space the band above it leaves free. */
.plex {
  position: relative;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  overflow: hidden;
}

.plex__image {
  flex: 1;
  min-block-size: 0;
}

.plex__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
}
</style>
