<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import type { Disposable, FileSelectEvent, VaultChangeEvent } from '@numen/plugin-sdk'
import type { FileTreeEntry, FileTreeViewProps } from './types'
import { WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(defineProps<FileTreeViewProps>(), {
  context: undefined,
  initialPath: '',
  entries: () => [],
})

const emit = defineEmits<{
  select: [entry: FileTreeEntry]
}>()

/* --------------------------------- State ---------------------------------- */
const selectedPath = ref(props.initialPath)
const expandedPaths = ref<Set<string>>(new Set())
const cleanupDisposables: Disposable[] = []

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (!props.context) return

  const vaultSub = props.context.onVaultChange((_event: VaultChangeEvent) => {
    // Reload or refresh tree on vault changes
  })
  cleanupDisposables.push(vaultSub)

  const fileSub = props.context.onFileSelect((event: FileSelectEvent) => {
    selectedPath.value = event.path
  })
  cleanupDisposables.push(fileSub)
})

onUnmounted(() => {
  for (const d of cleanupDisposables) {
    d.dispose()
  }
  cleanupDisposables.length = 0
})

/* -------------------------------- Handlers -------------------------------- */
function onSelectEntry(entry: FileTreeEntry) {
  selectedPath.value = entry.path
  emit('select', entry)
  if (props.context) {
    props.context.emitFileSelect({ path: entry.path, title: entry.name })
  }
}

function onToggleFolder(path: string) {
  if (expandedPaths.value.has(path)) {
    expandedPaths.value.delete(path)
  } else {
    expandedPaths.value.add(path)
  }
}

/* -------------------------------- Helpers --------------------------------- */
function isFolderExpanded(path: string): boolean {
  return expandedPaths.value.has(path)
}
</script>

<template>
  <div class="file-tree-view">
    <div v-if="props.entries.length === 0" class="file-tree-view__empty">
      {{ words.empty }}
    </div>
    <ul v-else class="file-tree-view__list">
      <li
        v-for="entry in props.entries"
        :key="entry.path"
        :class="[
          'file-tree-view__item',
          { 'file-tree-view__item--selected': entry.path === selectedPath },
        ]"
        @click="onSelectEntry(entry)"
      >
        <button
          v-if="entry.isFolder"
          type="button"
          class="file-tree-view__toggle"
          @click.stop="onToggleFolder(entry.path)"
        >
          {{ isFolderExpanded(entry.path) ? '▼' : '▶' }}
        </button>
        <span class="file-tree-view__name">{{ entry.name }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.file-tree-view {
  display: flex;
  flex-direction: column;
  block-size: 100%;
}

.file-tree-view__empty {
  padding: 1rem;
  opacity: 0.6;
}

.file-tree-view__list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.file-tree-view__item {
  display: flex;
  align-items: center;
  padding: 0.25rem 0.5rem;
  cursor: pointer;
}

.file-tree-view__item--selected {
  background: var(--surface-selected, rgba(0, 0, 0, 0.05));
}

.file-tree-view__toggle {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0 0.25rem;
  margin-right: 0.25rem;
}

.file-tree-view__name {
  flex: 1;
}
</style>
