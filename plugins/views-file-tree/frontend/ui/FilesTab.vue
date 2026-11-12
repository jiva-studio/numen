<script setup lang="ts">
/**
 * Displays the vault file tree and handles file navigation and organisation gestures.
 */
import { computed } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Position } from '@numen/ui'

import type { DropPosition, FilesTabState } from '../types'
import { getRows } from '../lib/rows'
import { getDragTypes, readDroppedData } from '../lib/drag'
import { useFilesTabView } from '../model/useFilesTabView'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ state: FilesTabState }>()

/* --------------------------------- State ---------------------------------- */
const rows = computed(() => getRows(props.state.list.queries.rows.value))

const { dropTarget, renamingPath, items, getRowIcon, getItemIcon } = useFilesTabView(
  () => props.state,
)

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
