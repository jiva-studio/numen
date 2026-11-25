<script setup lang="ts">
/**
 * Displays the graph of notes and handles graph interaction gestures.
 */
import { computed } from 'vue'
import { Inbox, Plus } from '@lucide/vue'
import { Button, Menu, LoadingIndicator } from '@numen/ui'
import { Plex, getInputStrategies, hasTouchPointer, useTypeSize } from '../plex'
import type { MenuTrigger } from '@numen/ui'
import PlexQuickLinkPopover from './PlexQuickLinkPopover.vue'
import PlexBoardSlot from './PlexBoardSlot.vue'
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
const { look } = props.state

const type = useTypeSize()
const input = getInputStrategies(hasTouchPointer())
const boardProps = computed(() => {
  const { board } = props.state.canvas.widgets
  return { board: board.key, path: board.here.value, go: board.go }
})

const options = computed(() => getPlexOptions(look, type.value))

const {
  rootElement,
  picture,
  lastPointer,
  onPointerDown,
  onPointerUp,
  onContextMenu,
  onEdgeMenu,
  closeMenu,
} = usePlexCanvas({ state: props.state })

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

function onDismissImage() {
  props.state.menu.dismiss()
}

function onChooseMenuItem(id: string) {
  closeMenu(id)
}

function onCreateFirstNote() {
  props.state.menu.makeNote().catch(props.state.reportFailure)
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
  <div
    ref="rootElement"
    class="plex"
    @contextmenu="onContextMenu"
    @pointerdown.capture="onPointerDown"
    @pointerup.capture="onPointerUp"
  >
    <p v-if="props.state.canvas.view.error.value" class="caution">
      {{ props.state.canvas.view.error.value }}
    </p>

    <PlexBoardSlot
      place="top"
      :entries="props.state.canvas.widgets.boardTop.value"
      v-bind="boardProps"
    />

    <!-- Until the first graph lands, an empty plex and one no one has
         answered for yet are drawn the same way. -->
    <div v-if="props.state.canvas.empty.value" class="plex__empty">
      <Inbox class="plex__empty-icon" aria-hidden="true" />
      <p class="plex__empty-title">{{ words.emptyTitle }}</p>
      <p class="plex__empty-hint">{{ words.emptyHint }}</p>
      <Button class="plex__empty-create" @click="onCreateFirstNote">
        <Plus class="plex__icon" aria-hidden="true" />
        {{ words.emptyCreate }}
      </Button>
    </div>

    <LoadingIndicator v-else-if="props.state.canvas.view.isLoading.value" :label="words.loading" />

    <Plex
      v-else-if="graph"
      ref="picture"
      class="plex__image"
      :graph="graph!"
      :options="options"
      :show-pills="look.showPills.value"
      :highlight-links="look.highlightLinks.value"
      :reaching="input.reaching"
      :showing="input.showing"
      :duration="look.animationLength.value"
      :line-width="look.lineThickness.value"
      :outline-width="look.outlineThickness.value"
      :creatable="props.state.canvas.creatable"
      :pending="pending"
      :dragged="dragged"
      :drop-name="words.dropName"
      :widgets-above="props.state.canvas.widgets.above.value"
      :widgets-below="props.state.canvas.widgets.below.value"
      :widgets-over="props.state.canvas.widgets.over.value"
      :get-path="props.state.canvas.widgets.getPath"
      @activate="onActivateNode"
      @create="onCreateNode"
      @link="onLinkNodes"
      @bring="onBringNodes"
      @menu="onOpenMenu"
      @show="onShowNode"
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

    <PlexBoardSlot
      place="bottom"
      :entries="props.state.canvas.widgets.boardBottom.value"
      v-bind="boardProps"
    />

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

.plex__empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-family: var(--numen-font-sans);
  text-align: center;
}

.plex__empty-icon {
  inline-size: 2.5rem;
  block-size: 2.5rem;
  margin-block-end: 0.5rem;
  color: var(--numen-hushed);
  stroke-width: 1.5;
}

/* The plus has room inside its own box, so the side before it takes less padding to look even with the
   side after the text. */
.plex__empty-create {
  margin-block-start: 0.75rem;
  padding-inline-start: 0.625rem;
}

.plex__empty-title {
  margin: 0;
  color: var(--numen-ink);
  font-weight: 600;
}

.plex__empty-hint {
  margin: 0;
  color: var(--numen-hushed);
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
