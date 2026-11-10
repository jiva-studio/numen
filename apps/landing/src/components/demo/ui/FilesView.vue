<script setup lang="ts">
/**
 * The folders and files of the demo vault: a tree that opens, selects and renames in place, and the menu
 * of a row.
 */
import { computed } from 'vue'
import { Menu, Tree, type Position } from '@numen/ui'
import type { DropTarget, Files } from '../model/files'
import type { Vault } from '../model/vault'
import { getCommandIcon, getEntryIcon } from './icons'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ files: Files; vault: Vault }>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  run: [command: string, target: { path: string; others: readonly string[] }]
}>()

/* --------------------------------- State ---------------------------------- */
const renaming = computed({
  get: () => props.files.renaming.value,
  set: (row: string | null) => {
    props.files.setRenaming(row)
  },
})
const items = computed(() => props.files.getMenuItems(props.files.menu.value?.path ?? null))

/* -------------------------------- Handlers -------------------------------- */
function onOpenMenu(row: string | null, at: Position) {
  props.files.openMenu(row, at)
}

function onChooseItem(id: string) {
  emit('run', id, props.files.getMenuTarget())
  props.files.closeMenu()
}

function onMoveRows(rows: readonly string[], target: DropTarget) {
  props.files.moveRows(rows, target)
}

/* -------------------------------- Helpers --------------------------------- */
const getRowIcon = (id: string, isOpen: boolean) =>
  getEntryIcon(props.vault.getEntryAt(id), isOpen)
</script>

<template>
  <div class="files">
    <Tree
      v-model:renaming-path="renaming"
      class="files__tree"
      name="Files"
      :rows="files.rows.value"
      :open="files.openRows.value"
      :selected="files.selected.value"
      @open="files.openFolder"
      @close="files.closeFolder"
      @select="files.select"
      @activate="files.activate"
      @rename="files.rename"
      @move="onMoveRows"
      @remove="files.remove"
      @menu="onOpenMenu"
    >
      <template #icon="{ id, isOpen }">
        <component :is="getRowIcon(id, isOpen)" class="files__icon" aria-hidden="true" />
      </template>
    </Tree>

    <Menu
      v-if="files.menu.value"
      :items="items"
      :at="files.menu.value.at"
      is-open
      @choose="onChooseItem"
      @dismiss="files.closeMenu"
    >
      <template #icon="{ id }">
        <component :is="getCommandIcon(id)" v-if="getCommandIcon(id)" class="files__icon" aria-hidden="true" />
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
