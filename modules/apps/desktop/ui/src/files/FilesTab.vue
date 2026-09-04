<script setup lang="ts">
/**
 * A files tab: the tree of the vault, the menu on a row of it, and what this
 * tab could not read.
 *
 * A row's identity is the path the vault files it under, and what is drawn
 * beside a name is what the vault holds there. A row also names the folder a
 * file dropped on it from outside the window is filed in.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Marking, Point, Row as TreeRow } from '@numen/ui'
import {
  Book,
  File,
  FileText,
  Folder,
  FolderOpen,
  Layers,
  LayoutTemplate,
  type LucideIcon,
} from '@lucide/vue'
import Caution from '../Caution.vue'
import type { NoteType, Source } from '../core'
import { iconFor, iconOfNote } from '../icons'
import type { Dropped, Held } from './kind'
import type { Row } from './listing'
import { itemsFor } from './menu'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

/**
 * The mark the window's drag and drop reads: the attribute it looks a target
 * up by, carrying the folder a file let go there is filed in. The window puts
 * `file-drop-target-active` on whichever it is over.
 */
const dropTarget = computed<Marking>(() => ({
  attribute: 'data-file-drop-target',
  valueFor: (row: string | null) => props.held.folderFor(row),
}))

/** The tree as the component takes it, a path standing for each row. */
const drawn = (rows: readonly Row[]): TreeRow[] =>
  rows.map((one) => ({
    id: one.entry.path,
    name: one.entry.displayName,
    holds: one.entry.folder,
    rows: drawn(one.rows),
  }))

const rows = computed(() => drawn(props.held.list.rows.value))

/** The row whose name is in a field, which the tree opens and closes itself. */
const renaming = computed({
  get: () => props.held.renaming.value,
  set: (row: string | null) => {
    props.held.renaming.value = row
  },
})

/** What the vault holds at a row: a folder, a note of one of three kinds, or a file. */
const kindOf = (id: string): Source | NoteType | 'folder' => {
  const entry = props.held.list.entryAt(id)
  if (!entry) return 'other'
  if (entry.folder) return 'folder'
  return entry.kind === 'note' ? entry.type : entry.kind
}

/**
 * The icon drawn beside a name. A folder says whether what it holds is drawn:
 * the tree draws nothing else for it.
 */
const entryIcon = (id: string, open: boolean): LucideIcon => {
  const kind = kindOf(id)
  if (kind === 'folder') return open ? FolderOpen : Folder
  if (kind === 'book') return Book
  if (kind === 'note' || kind === 'deck' || kind === 'stencil' || kind === 'preset') {
    return iconOfNote(kind)
  }
  return File
}

/** What the menu offers: on the row it was asked for on, or off every row. */
const items = computed(() => {
  const asked = props.held.menu.value
  if (!asked || asked.path === null) return itemsFor(null, false, props.held.canRun)

  const entry = props.held.list.entryAt(asked.path)
  const on = { source: entry?.kind ?? 'other', folder: entry?.folder ?? false }
  return itemsFor(on, props.held.over(asked.path).length > 1, props.held.canRun)
})

/**
 * A file the vault holds no source for is not reported by the watcher, so the
 * tree is read again whenever the window comes back to the front.
 */
onMounted(() => globalThis.addEventListener('focus', props.held.list.again))
onUnmounted(() => globalThis.removeEventListener('focus', props.held.list.again))
</script>

<template>
  <div class="files">
    <Caution v-if="props.held.list.trouble.value">
      {{ props.held.list.trouble.value }}
    </Caution>

    <Tree
      v-model:renaming="renaming"
      class="files__tree"
      :rows="rows"
      :open="props.held.list.openRows.value"
      :selected="props.held.list.chosen.value"
      :name="words.tree"
      :counted="words.carrying"
      :marking="dropTarget"
      @open="(row: string) => props.held.open(row)"
      @close="(row: string) => props.held.close(row)"
      @select="(rows: readonly string[]) => props.held.select(rows)"
      @activate="(row: string) => props.held.activate(row)"
      @rename="(row: string, name: string) => void props.held.rename(row, name)"
      @move="(rows: readonly string[], at: Dropped) => void props.held.move(rows, at)"
      @carry="(rows: readonly string[]) => props.held.carry(rows)"
      @drop="props.held.drop()"
      @remove="(rows: readonly string[]) => props.held.remove(rows)"
      @menu="(row: string | null, at: Point) => props.held.asks({ path: row, at })"
    >
      <template #icon="{ id, open }">
        <component :is="entryIcon(id, open)" class="files__icon" aria-hidden="true" />
      </template>

      <template #silence>{{ words.empty }}</template>
    </Tree>

    <Menu
      v-if="props.held.menu.value"
      :items="items"
      :at="props.held.menu.value!.at"
      open
      @choose="(id: string) => props.held.chose(id)"
      @dismiss="props.held.dismiss()"
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

/* Where a file carried in from outside would land. The window puts this class
   on the element under the pointer for as long as the drag is over it. */
.files__tree :deep(.file-drop-target-active),
.files__tree.file-drop-target-active {
  box-shadow: inset 0 0 0 var(--numen-ring-width) var(--numen-ring);
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.files__icon {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  stroke-width: 1.875;
  opacity: 0.75;
}
</style>
