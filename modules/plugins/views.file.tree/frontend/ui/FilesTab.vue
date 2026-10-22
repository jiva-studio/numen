<script setup lang="ts">
/**
 * Displays the vault file tree and handles file navigation and organisation gestures.
 */
import { computed, onMounted, onUnmounted, type Component } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Position, RowMarker } from '@numen/ui'

import type { DropPosition, FilesTabState } from '../types'
import { getRows } from '../lib/rows'
import { getDroppedUrls, hasUrl } from '../lib/drag'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: FilesTabState }>()

/* --------------------------------- State ---------------------------------- */
/**
 * The mark the window's drag and drop reads: the attribute it looks a target
 * up by, carrying the folder a file let go there is filed in.
 */
const dropTarget = computed<RowMarker>(() => ({
  attribute: 'data-file-drop-target',
  getAttributeValue: (row: string | null) => props.state.getFolderFor(row),
}))

const rows = computed(() => getRows(props.state.list.rows.value))

/** The row whose name is in a field, which the tree opens and closes itself. */
const renamingPath = computed({
  get: () => props.state.renamingPath.value,
  set: (row: string | null) => props.state.setRenamingPath(row),
})

/** What the menu offers: on the row it was asked for on, or off every row. */
const items = computed(() => props.state.getMenuItems(props.state.menu.value?.path ?? null))

/* --------------------------------- Hooks ---------------------------------- */
const refreshTree = () => void props.state.list.refresh()
onMounted(() => globalThis.addEventListener('focus', refreshTree))
onUnmounted(() => globalThis.removeEventListener('focus', refreshTree))

/* -------------------------------- Handlers -------------------------------- */
function onDrop(event: DragEvent) {
  const urls = getDroppedUrls(event.dataTransfer)
  if (urls.length === 0) return
  event.preventDefault()
  void props.state.importUrls(urls)
}

function onDragOver(event: DragEvent) {
  if (hasUrl(event.dataTransfer?.types)) event.preventDefault()
}

function onOpenEntry(row: string) {
  props.state.open(row)
}

function onCloseEntry(row: string) {
  props.state.close(row)
}

function onSelectEntries(rows: readonly string[]) {
  props.state.select(rows)
}

function onActivateEntry(row: string) {
  props.state.activate(row)
}

function onRenameEntry(row: string, name: string) {
  void props.state.rename(row, name)
}

function onMoveEntries(targetRows: readonly string[], target: DropPosition) {
  void props.state.move(targetRows, at)
}

function onDragEntries(rows: readonly string[]) {
  props.state.drag(rows)
}

function onDropEntries() {
  props.state.drop()
}

function onRemoveEntries(rows: readonly string[]) {
  props.state.remove(rows)
}

function onOpenMenu(row: string | null, position: Position) {
  props.state.openMenu({ path: row, at })
}

function onChooseMenuItem(id: string) {
  props.state.chooseMenuItem(id)
}

function onDismissMenu() {
  props.state.dismissMenu()
}

/* -------------------------------- Helpers --------------------------------- */
function getRowIcon(id: string, isOpen: boolean): Component {
  return props.state.getEntryIcon(props.state.list.getEntryAt(id), isOpen)
}

function getItemIcon(id: string): Component | null {
  return props.state.getCommandIcon(id)
}
</script>

<template>
  <div class="files" @dragover="onDragOver" @drop="onDrop">
    <p v-if="props.state.list.errorMessage.value" class="caution">
      {{ props.state.list.errorMessage.value }}
    </p>

    <Tree
      v-model:renaming-path="renamingPath"
      class="files__tree"
      :rows="rows"
      :open="props.state.list.openRows.value"
      :selected="props.state.list.selectedPaths.value"
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

      <template #silence>{{ words.empty }}</template>
    </Tree>

    <Menu
      v-if="props.state.menu.value"
      :items="items"
      :at="props.state.menu.value!.at"
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
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
