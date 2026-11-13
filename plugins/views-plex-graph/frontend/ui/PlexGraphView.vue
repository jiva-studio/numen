<script setup lang="ts">
/**
 * Displays the graph of notes and handles graph interaction gestures.
 */
import { computed } from 'vue'
import { Menu, LoadingIndicator } from '@numen/ui'
import { Plex, useTypeSize } from '../plex'
import type { MenuTrigger } from '@numen/ui'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'
import PlexLinkInspectorPopover from './PlexLinkInspectorPopover.vue'
import type { Connection } from '@numen/plugin-sdk'
import type { PlexTabState, PlexRelation, PlexDestination } from '../types'
import { WORDS as words } from '../words'
import { usePlexCanvas } from '../model/usePlexCanvas'
import { getPlexOptions } from '../model/plexOptions'

/* ----------------------------- Props & Emits ------------------------------ */
defineOptions({ name: 'PlexTab' })
const props = defineProps<{ state: PlexTabState }>()

/* --------------------------------- State ---------------------------------- */
const { dragged, picture: graph } = props.state.canvas
const { request: menu, menuItems: items, getMenuIcon, getNodeIcon } = props.state.menu
const { quickLink } = props.state.quickLink
const { linkInspector } = props.state.inspector
const { maxHeadings } = props.state.headings
const { look } = props.state

const type = useTypeSize()
const options = computed(() => getPlexOptions(look, type.value, maxHeadings.value))

const { rootElement, picture, lastPointer, onPointerUp, onContextMenu, onEdgeMenu, closeMenu } =
  usePlexCanvas({ state: props.state })

const pending = computed(() =>
  quickLink.value ? { from: quickLink.value.from, at: quickLink.value.at } : null,
)

/* -------------------------------- Handlers -------------------------------- */
function onActivateNode(node: string) {
  props.state.nodes.activate(node)
}

function onCreateNode(from: string, relation: PlexRelation, at?: { x: number; y: number }) {
  props.state.nodes
    .addNode(from, relation, at ?? lastPointer.value)
    .catch(props.state.reportFailure)
}

function onLinkNodes(from: string, to: string, relation: PlexRelation) {
  props.state.nodes.joinNodes(from, to, relation).catch(props.state.reportFailure)
}

function onBringNodes(nodes: readonly string[], relation: PlexRelation) {
  props.state.nodes.dropNodes(nodes, relation).catch(props.state.reportFailure)
}

function onOpenMenu(node: string, position: { x: number; y: number }, opening: MenuTrigger) {
  props.state.menu.openMenu({ node, at: position, opening })
}

function onShowNode(node: string, how: PlexDestination) {
  props.state.nodes.openNode(node, how)
}

function onEnterHeading(node: string, heading: string) {
  props.state.nodes.openHeading(node, heading)
}

function onDismissImage() {
  props.state.menu.dismiss()
}

function onChooseMenuItem(id: string) {
  closeMenu(id)
}

function onDismissMenu() {
  closeMenu()
}

function onSelectQuickLinkNote(path: string) {
  props.state.quickLink.confirmQuickLink(path, true).catch(props.state.reportFailure)
}

function onCreateQuickLinkNote(title: string) {
  props.state.quickLink.confirmQuickLink(title, false).catch(props.state.reportFailure)
}

function onDismissQuickLink() {
  props.state.quickLink.dismissQuickLink()
}

function onSaveLinkInspector(connection: Connection) {
  props.state.inspector.saveLinkInspector(connection).catch(props.state.reportFailure)
}

function onRemoveLinkInspector() {
  props.state.inspector.removeLinkInspector().catch(props.state.reportFailure)
}

function onDismissLinkInspector() {
  props.state.inspector.dismissLinkInspector()
}
</script>

<template>
  <div ref="rootElement" class="plex" @contextmenu="onContextMenu" @pointerup.capture="onPointerUp">
    <p v-if="props.state.canvas.view.error.value" class="caution">
      {{ props.state.canvas.view.error.value }}
    </p>

    <!-- Until the first graph lands, an empty plex and one no one has
         answered for yet are drawn the same way. -->
    <LoadingIndicator v-if="props.state.canvas.view.isLoading.value" :label="words.loading" />

    <Plex
      v-else-if="graph"
      ref="picture"
      class="plex__image"
      :graph="graph!"
      :options="options"
      :show-pills="look.showPills.value"
      :duration="look.animationLength.value"
      :line-width="look.lineThickness.value"
      :outline-width="look.outlineThickness.value"
      :creatable="props.state.canvas.creatable"
      :pending="pending"
      :dragged="dragged"
      :drop-name="words.dropName"
      :headings="props.state.headings.getHeadings"
      :names="props.state.names.getNames"
      :show-names="props.state.names.showNames.value"
      @activate="onActivateNode"
      @create="onCreateNode"
      @link="onLinkNodes"
      @bring="onBringNodes"
      @menu="onOpenMenu"
      @show="onShowNode"
      @enter="onEnterHeading"
      @choose-name="props.state.nodes.chooseName"
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
      :search="props.state.quickLink.searchNotes"
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
  --icon-inline-size: 0.875rem;
  --icon-block-size: 0.875rem;
  inline-size: var(--icon-inline-size);
  block-size: var(--icon-block-size);
  stroke-width: 1.875;
}
</style>
