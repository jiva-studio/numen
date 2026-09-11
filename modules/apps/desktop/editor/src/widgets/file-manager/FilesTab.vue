<script setup lang="ts">
/**
 * Displays the vault file tree and handles file navigation and organisation gestures.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Position, Row, RowMarker } from '@numen/ui'
import type { LucideIcon } from '@lucide/vue'

import { iconFor, iconOfEntry } from '../../shared/icons'
import type { DropPosition, FilesTabState } from './types'
import type { ListingRow } from './listing'
import { addressDropped, carriesAddress } from './drag'
import { itemsFor } from './menu'
import { WORDS as words } from './words'

// --- Props & Emits ---
const props = defineProps<{ state: FilesTabState }>()

// --- State ---
/**
 * The mark the window's drag and drop reads: the attribute it looks a target
 * up by, carrying the folder a file let go there is filed in. The window puts
 * `file-drop-target-active` on whichever it is over.
 */
const dropTarget = computed<RowMarker>(() => ({
  attribute: 'data-file-drop-target',
  valueFor: (row: string | null) => props.state.folderFor(row),
}))

const rows = computed(() => drawn(props.state.list.rows.value))

/** The row whose name is in a field, which the tree opens and closes itself. */
const renaming = computed({
  get: () => props.state.renaming.value,
  set: (row: string | null) => props.state.setRenamingPath(row),
})

/** What the menu offers: on the row it was asked for on, or off every row. */
const items = computed(() => {
  const asked = props.state.menu.value
  if (!asked || asked.path === null) return itemsFor(null, false, props.state.canRun)

  const entry = props.state.list.entryAt(asked.path)
  const on = {
    source: entry?.kind ?? 'other',
    folder: entry?.folder ?? false,
  }
  return itemsFor(on, props.state.over(asked.path).length > 1, props.state.canRun)
})

// --- Handlers ---
/** An address dragged out of a browser, made into the file it is kept in. */
function onDrop(event: DragEvent) {
  const address = addressDropped(event.dataTransfer)
  if (!address) return
  event.preventDefault()
  void (props.state.importAddress ?? props.state.imports)(address)
}

/** A drag carrying an address is one this tab takes. */
function onDragOver(event: DragEvent) {
  if (carriesAddress(event.dataTransfer?.types)) event.preventDefault()
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

function onMoveEntries(rows: readonly string[], at: DropPosition) {
  void props.state.move(rows, at)
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

function onOpenMenu(row: string | null, at: Position) {
  props.state.openMenu({ path: row, at })
}

function onChooseMenuItem(id: string) {
  props.state.chooseMenuItem(id)
}

function onDismissMenu() {
  props.state.dismiss()
}

// --- Helpers ---
/** The tree as the component takes it, a path standing for each row. */
function drawn(rows: readonly ListingRow[]): Row[] {
  return rows.map((one) => ({
    id: one.entry.path,
    name: one.entry.name,
    holds: one.entry.folder,
    rows: drawn(one.rows),
  }))
}

/** The icon drawn beside a name, for whatever the vault holds at that row. */
function entryIcon(id: string, open: boolean): LucideIcon {
  return iconOfEntry(props.state.list.entryAt(id), open)
}

/**
 * A file the vault holds no source for is not reported by the watcher, so the
 * tree is read again whenever the window comes back to the front.
 */
const again = () => void props.state.list.again()
onMounted(() => globalThis.addEventListener('focus', again))
onUnmounted(() => globalThis.removeEventListener('focus', again))
</script>

<template>
  <div class="files" @dragover="onDragOver" @drop="onDrop">
    <p v-if="props.state.list.error.value" class="caution">
      {{ props.state.list.error.value }}
    </p>

    <Tree
      v-model:renaming="renaming"
      class="files__tree"
      :rows="rows"
      :open="props.state.list.openRows.value"
      :selected="props.state.list.chosen.value"
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
/* The tree takes what the band above it leaves. */
.files {
  display: flex;
  flex-direction: column;
  block-size: 100%;
}

.files__tree {
  flex: 1;
  min-block-size: 0;
}

/* Where a file dragged in from outside would land. The window puts this class
   on the element under the pointer for as long as the drag is over it. */
.files__tree :deep(.file-drop-target-active),
.files__tree.file-drop-target-active {
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.files__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
