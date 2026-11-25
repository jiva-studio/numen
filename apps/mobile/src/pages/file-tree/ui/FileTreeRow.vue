<script setup lang="ts">
/** One file or folder of the tree: a button to open it, and, for a downloaded file, a button to remove its download. */
import type { TreeRow } from '../model/useFileTree'

defineProps<{ row: TreeRow }>()

const emit = defineEmits<{ tap: []; 'drop-download': [] }>()
</script>

<template>
  <li class="file-tree-row" data-testid="file-row">
    <button type="button" class="file-tree-row__tap" data-testid="row-tap" @click="emit('tap')">
      <span>{{ row.name }}</span>
      <span v-if="row.isDownloading">Downloading</span>
      <span v-else-if="row.isAbsent">Not downloaded</span>
    </button>
    <button
      v-if="!row.isFolder && !row.isAbsent && !row.isDownloading"
      type="button"
      data-testid="drop-download"
      @click="emit('drop-download')"
    >
      Remove download
    </button>
  </li>
</template>

<style scoped>
.file-tree-row {
  display: flex;
  align-items: center;
  gap: var(--numen-gutter);
}

.file-tree-row__tap {
  display: flex;
  flex: 1;
  align-items: center;
  gap: var(--numen-gutter);
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-align: start;
}
</style>
