<script setup lang="ts">
/**
 * A files tab: the tree of the vault, the menu on a row of it, and what this
 * tab could not read.
 *
 * The tree takes names and hands identities back, and a row's identity here is
 * the path the vault files it under. Every gesture is handed to what the tab
 * holds. What is drawn beside a name is what the vault holds there.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Menu, Tree } from '@numen/ui'
import type { Point, Row as TreeRow } from '@numen/ui'
import type { Source } from '../core'
import type { Dropped, Held } from './kind'
import type { Row } from './listing'
import { itemsFor } from './menu'
import { WORDS as words } from './words'

const props = defineProps<{ held: Held }>()

/** The tree as the component takes it, a path standing for each row. */
const drawn = (rows: readonly Row[]): TreeRow[] =>
  rows.map((one) => ({
    id: one.entry.path,
    name: one.entry.name,
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

/** What the vault holds at a row, which is the mark drawn beside its name. */
const markOf = (id: string): Source | 'folder' => {
  const entry = props.held.list.entryAt(id)
  if (!entry) return 'other'
  return entry.folder ? 'folder' : entry.kind
}

/** What the menu offers: on the row it was asked for on, or off every row. */
const items = computed(() => {
  const asked = props.held.menu.value
  if (!asked || asked.path === null) return itemsFor(null, false)

  const entry = props.held.list.entryAt(asked.path)
  const on = { source: entry?.kind ?? 'other', folder: entry?.folder ?? false }
  return itemsFor(on, props.held.over(asked.path).length > 1)
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
    <p v-if="props.held.list.trouble.value" class="warning">
      {{ props.held.list.trouble.value }}
    </p>

    <Tree
      v-model:renaming="renaming"
      class="files__tree"
      :rows="rows"
      :open="props.held.list.openRows.value"
      :selected="props.held.list.chosen.value"
      :name="words.tree"
      :counted="words.carrying"
      @open="(row: string) => props.held.open(row)"
      @close="(row: string) => props.held.close(row)"
      @select="(rows: readonly string[]) => props.held.select(rows)"
      @activate="(row: string) => props.held.activate(row)"
      @rename="(row: string, name: string) => void props.held.rename(row, name)"
      @move="(rows: readonly string[], at: Dropped) => void props.held.move(rows, at)"
      @remove="(rows: readonly string[]) => props.held.remove(rows)"
      @menu="(row: string | null, at: Point) => props.held.asks({ path: row, at })"
    >
      <template #icon="{ id }">
        <svg class="files__mark" viewBox="0 0 16 16" aria-hidden="true">
          <path v-if="markOf(id) === 'folder'" d="M1 4h5l1.5 2H15v7H1z" />
          <path v-else-if="markOf(id) === 'book'" d="M3 2h10v12H3zM5 5h6M5 8h6" />
          <path v-else d="M4 1h5l3 3v11H4zM9 1v3h3" />
          <circle v-if="markOf(id) === 'note'" cx="8" cy="10" r="2" />
        </svg>
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
    />
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

.files__mark {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.25;
  stroke-linejoin: round;
  opacity: 0.75;
}

/* A warning carries a filesystem path, and a long one breaks where it stands. */
.warning {
  margin: 0;
  padding: 0.4rem 1rem;
  font-family: var(--numen-font-sans);
  font-size: 0.8rem;
  background: var(--numen-caution-bg);
  color: var(--numen-caution-fg);
  overflow-wrap: break-word;
}
</style>
