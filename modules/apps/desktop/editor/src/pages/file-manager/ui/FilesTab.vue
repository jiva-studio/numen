<script setup lang="ts">
/**
 * Displays the vault file tree and handles file navigation and organisation gestures.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Position, Row, RowMarker } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'

import { iconFor } from '@/shared/icons'
import { iconOfEntry } from '../lib/icons'
import type { DropPosition, FilesTabState, ListingRow } from '../types'
import { addressDropped, carriesAddress } from '../lib/drag'
import { itemsFor } from '../lib/menu'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: FilesTabState }>()

// --- State ---
/**
 * The mark the window's drag and drop reads: the attribute it looks a target
 * up by, carrying the folder a file let go there is filed in.
 */
const dropTarget = computed<RowMarker>(() => ({
  attribute: 'data-file-drop-target',
  valueFor: (row: string | null) => props.state.getFolderFor(row),
}))

const rows = computed(() => rowsOf(props.state.list.rows.value))

/** The row whose name is in a field, which the tree opens and closes itself. */
const renaming = computed({
  get: () => props.state.renamingPath.value,
  set: (row: string | null) => props.state.setRenamingPath(row),
})

/** What the menu offers: on the row it was asked for on, or off every row. */
const items = computed(() => {
  const asked = props.state.menu.value
  if (!asked || asked.path === null) return itemsFor(null, false, props.state.canRun)

  const entry = props.state.list.getEntryAt(asked.path)
  const on = {
    source: entry?.kind ?? 'other',
    folder: entry?.folder ?? false,
  }
  return itemsFor(on, props.state.getOverPaths(asked.path).length > 1, props.state.canRun)
})

// --- Handlers ---
function onDrop(event: DragEvent) {
  const address = addressDropped(event.dataTransfer)
  if (!address) return
  event.preventDefault()
  void props.state.importAddress(address)
}

function onDragOver(event: DragEvent) {
  if (carriesAddress(event.dataTransfer?.types)) event.preventDefault()
}

function onOpenEntry(row: string) {
  props.state.open(row)
}

function onCloseEntry(row: string) {
  props.state.close(row)
}

function onSelectEntries(selectedRows: readonly string[]) {
  props.state.select(selectedRows)
}

function onActivateEntry(row: string) {
  props.state.activate(row)
}

function onRenameEntry(row: string, name: string) {
  void props.state.rename(row, name)
}

function onMoveEntries(targetRows: readonly string[], at: DropPosition) {
  void props.state.move(targetRows, at)
}

function onDragEntries(draggedRows: readonly string[]) {
  props.state.drag(draggedRows)
}

function onDropEntries() {
  props.state.drop()
}

function onRemoveEntries(removedRows: readonly string[]) {
  props.state.remove(removedRows)
}

function onOpenMenu(row: string | null, at: Position) {
  props.state.openMenu({ path: row, at })
}

function onChooseMenuItem(id: string) {
  props.state.chooseMenuItem(id)
}

function onDismissMenu() {
  props.state.dismissMenu()
}

// --- Helpers ---
function rowsOf(listingRows: readonly ListingRow[]): Row[] {
  return listingRows.map((one) => ({
    id: one.entry.path,
    name: one.entry.name,
    holds: one.entry.folder,
    rows: rowsOf(one.rows),
  }))
}

function entryIcon(id: string, open: boolean): LucideIcon {
  return iconOfEntry(props.state.list.getEntryAt(id), open)
}

const refreshTree = () => void props.state.list.refresh()
onMounted(() => globalThis.addEventListener('focus', refreshTree))
onUnmounted(() => globalThis.removeEventListener('focus', refreshTree))
</script>

<template>
  <div class="files" @dragover="onDragOver" @drop="onDrop">
    <p v-if="props.state.list.errorMessage.value" class="caution">
      {{ props.state.list.errorMessage.value }}
    </p>

    <Tree
      v-model:renaming="renaming"
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
      <template #icon="{ id, open }">
        <component :is="entryIcon(id, open)" class="files__icon" aria-hidden="true" />
      </template>

      <template #silence>{{ words.empty }}</template>
    </Tree>

    <Menu
      v-if="props.state.menu.value"
      :items="items"
      :at="props.state.menu.value!.at"
      open
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    >
      <template #icon="{ id }">
        <component :is="iconFor(id)" v-if="iconFor(id)" class="files__icon" aria-hidden="true" />
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
