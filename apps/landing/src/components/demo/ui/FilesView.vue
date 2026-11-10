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
  get: () => props.files.state.renaming.value,
  set: (row: string | null) => {
    props.files.edits.setRenaming(row)
  },
})
const items = computed(() => props.files.menus.getMenuItems(props.files.state.menu.value?.path ?? null))

/* -------------------------------- Handlers -------------------------------- */
function onOpenMenu(row: string | null, at: Position) {
  props.files.menus.openMenu(row, at)
}

function onChooseItem(id: string) {
  emit('run', id, props.files.menus.getMenuTarget())
  props.files.menus.closeMenu()
}

function onMoveRows(rows: readonly string[], target: DropTarget) {
  props.files.edits.moveRows(rows, target)
}

/* -------------------------------- Helpers --------------------------------- */
const getRowIcon = (id: string, isOpen: boolean) =>
  getEntryIcon(props.vault.read.getEntryAt(id), isOpen)
</script>

<template>
  <div class="files">
    <Tree
      v-model:renaming-path="renaming"
      class="files__tree"
      name="Files"
      :rows="files.state.rows.value"
      :open="files.state.openRows.value"
      :selected="files.state.selected.value"
      @open="files.tree.openFolder"
      @close="files.tree.closeFolder"
      @select="files.tree.select"
      @activate="files.tree.activate"
      @rename="files.edits.rename"
      @move="onMoveRows"
      @remove="files.edits.remove"
      @menu="onOpenMenu"
    >
      <template #icon="{ id, isOpen }">
        <component :is="getRowIcon(id, isOpen)" class="files__icon" aria-hidden="true" />
      </template>
    </Tree>

    <Menu
      v-if="files.state.menu.value"
      :items="items"
      :at="files.state.menu.value.at"
      is-open
      @choose="onChooseItem"
      @dismiss="files.menus.closeMenu"
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
