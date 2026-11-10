<script setup lang="ts">
/**
 * Displays the vault file tree and handles file navigation and organisation gestures.
 */
import { computed, onMounted, onUnmounted, type Component } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Position, RowMarker } from '@numen/ui'

import type { DropPosition, FilesTabState } from '../types'
import { getRows } from '../lib/rows'
import { getDragTypes, readDroppedData } from '../lib/drag'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: FilesTabState }>()

/* --------------------------------- State ---------------------------------- */
/**
 * The mark the window's drag and drop reads: the attribute it looks a target
 * up by, carrying the folder a file dropped there is filed in.
 */
const dropTarget = computed<RowMarker>(() => ({
  attribute: 'data-file-drop-target',
  getAttributeValue: (row: string | null) => props.state.transfer.getFolderFor(row),
}))

const rows = computed(() => getRows(props.state.list.queries.rows.value))

/** The row whose name is in a field, which the tree opens and closes itself. */
const renamingPath = computed({
  get: () => props.state.renaming.path.value,
  set: (row: string | null) => props.state.renaming.setPath(row),
})

/** The menu items for the row the menu was opened on, or for no row. */
const items = computed(() =>
  props.state.menu.getItems(props.state.menu.request.value?.path ?? null),
)

/* --------------------------------- Hooks ---------------------------------- */
const refreshTree = () => {
  props.state.list.sync.refresh().catch(props.state.reportFailure)
}
onMounted(() => globalThis.addEventListener('focus', refreshTree))
onUnmounted(() => globalThis.removeEventListener('focus', refreshTree))

/* -------------------------------- Handlers -------------------------------- */
function onDragOver(event: DragEvent) {
  if (props.state.transfer.canDropData(getDragTypes(event.dataTransfer))) event.preventDefault()
}

function onDrop(event: DragEvent) {
  const types = getDragTypes(event.dataTransfer)
  if (!props.state.transfer.canDropData(types)) return
  event.preventDefault()
  props.state.transfer
    .dropData(readDroppedData(event.dataTransfer))
    .catch(props.state.reportFailure)
}

function onOpenEntry(row: string) {
  props.state.entries.open(row)
}

function onCloseEntry(row: string) {
  props.state.entries.close(row)
}

function onSelectEntries(rows: readonly string[]) {
  props.state.entries.select(rows)
}

function onActivateEntry(row: string) {
  props.state.entries.activate(row)
}

function onRenameEntry(row: string, name: string) {
  props.state.entries.rename(row, name).catch(props.state.reportFailure)
}

function onMoveEntries(targetRows: readonly string[], target: DropPosition) {
  props.state.transfer.move(targetRows, target).catch(props.state.reportFailure)
}

function onDragEntries(rows: readonly string[]) {
  props.state.transfer.drag(rows)
}

function onDropEntries() {
  props.state.transfer.drop()
}

function onRemoveEntries(rows: readonly string[]) {
  props.state.entries.remove(rows)
}

function onOpenMenu(row: string | null, position: Position) {
  props.state.menu.open({ path: row, at: position })
}

function onChooseMenuItem(id: string) {
  props.state.menu.choose(id)
}

function onDismissMenu() {
  props.state.menu.dismiss()
}

/* -------------------------------- Helpers --------------------------------- */
function getRowIcon(id: string, isOpen: boolean): Component {
  return props.state.entries.getEntryIcon(props.state.list.queries.getEntryAt(id), isOpen)
}

function getItemIcon(id: string): Component | null {
  return props.state.menu.getIcon(id)
}
</script>

<template>
  <div class="files" @dragover="onDragOver" @drop="onDrop">
    <p v-if="props.state.list.state.errorMessage.value" class="caution">
      {{ props.state.list.state.errorMessage.value }}
    </p>

    <Tree
      v-model:renaming-path="renamingPath"
      class="files__tree"
      :rows="rows"
      :open="props.state.list.queries.openRows.value"
      :selected="props.state.list.state.selectedPaths.value"
      :name="words.tree"
      :counted="words.dragging"
      :marking="dropTarget"
      @open="onOpenEntry"
      @close="onCloseEntry"
      @select="onSelectEntries"
      @activate="onActivateEntry"
      @rename="onRenameEntry"
      @move="onMoveEntries"
      @drag="onDragEntries"
      @drop="onDropEntries"
      @remove="onRemoveEntries"
      @menu="onOpenMenu"
    >
      <template #icon="{ id, isOpen }">
        <component :is="getRowIcon(id, isOpen)" class="files__icon" aria-hidden="true" />
      </template>

      <template #silence>
        <template v-if="props.state.list.state.isLoaded.value">{{ words.empty }}</template>
      </template>
    </Tree>

    <Menu
      v-if="props.state.menu.request.value"
      :items="items"
      :at="props.state.menu.request.value!.at"
      is-open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component
          :is="getItemIcon(id)"
          v-if="getItemIcon(id)"
          class="files__icon"
          aria-hidden="true"
        />
      </template>
    </Menu>
  </div>
</template>

<style scoped>
.files {
  display: flex;
  flex-direction: column;
  block-size: 100%;
}

.files__tree {
  flex: 1;
  min-block-size: 0;
}

.files__icon {
  --icon-size: 0.875rem;
  inline-size: var(--icon-size);
  block-size: var(--icon-size);
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
