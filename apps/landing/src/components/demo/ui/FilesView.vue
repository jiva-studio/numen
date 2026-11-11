<script setup lang="ts">
/**
 * The folders and files of the vault the demo draws: a tree whose folders open and close and whose rows
 * are selected, and which changes nothing.
 */
import { ref } from 'vue'
import { Tree, type Row } from '@numen/ui'
import { FILE_ROWS, SELECTED_ROW } from '../lib/files'
import { getRowIcon } from './icons'

/* --------------------------------- State ---------------------------------- */
const getFolders = (rows: readonly Row[]): string[] =>
  rows.flatMap((row) => (row.hasChildren ? [row.id, ...getFolders(row.rows)] : []))
const open = ref<readonly string[]>(['', ...getFolders(FILE_ROWS)])
const selected = ref<readonly string[]>([SELECTED_ROW])

/* -------------------------------- Handlers -------------------------------- */
function onOpen(folder: string) {
  open.value = [...open.value, folder]
}

function onClose(folder: string) {
  open.value = open.value.filter((one) => one !== folder)
}

function onSelect(rows: readonly string[]) {
  selected.value = rows
}
</script>

<template>
  <div class="files">
    <Tree
      class="files__tree"
      name="Files"
      :rows="FILE_ROWS"
      :open="open"
      :selected="selected"
      @open="onOpen"
      @close="onClose"
      @select="onSelect"
    >
      <template #icon="{ id, isOpen }">
        <component :is="getRowIcon(id, isOpen)" class="files__icon" aria-hidden="true" />
      </template>
    </Tree>
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
