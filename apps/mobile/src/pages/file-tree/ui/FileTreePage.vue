<script setup lang="ts">
/**
 * The files of the vault, held and not downloaded. Tapping a file that is not downloaded downloads it and
 * opens it; a downloaded file opens at once. The page emits `open` with the path of the file to open.
 */
import { useFileTree } from '../model/useFileTree'

const emit = defineEmits<{ open: [path: string] }>()

const { rows, errorMessage, tap, dropDownload } = useFileTree((path) => emit('open', path))
</script>

<template>
  <section>
    <p v-if="errorMessage" role="alert" data-testid="error">{{ errorMessage }}</p>
    <ul class="file-tree__list">
      <li v-for="row in rows" :key="row.path" class="file-tree__row" data-testid="file-row">
        <button type="button" class="file-tree__tap" data-testid="row-tap" @click="tap(row)">
          <span>{{ row.name }}</span>
          <span v-if="row.isDownloading">Downloading</span>
          <span v-else-if="row.isAbsent">Not downloaded</span>
        </button>
        <button
          v-if="!row.isFolder && !row.isAbsent && !row.isDownloading"
          type="button"
          data-testid="drop-download"
          @click="dropDownload(row)"
        >
          Remove download
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.file-tree__list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.file-tree__row {
  display: flex;
  align-items: center;
  gap: var(--numen-gutter);
}

.file-tree__tap {
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
