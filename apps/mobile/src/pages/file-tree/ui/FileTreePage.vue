<script setup lang="ts">
/**
 * The files of the vault, held and not downloaded. Tapping a file that is not downloaded downloads it and
 * opens it; a downloaded file opens at once. The page emits `open` with the path of the file to open.
 */
import { useFileTree } from '../model/useFileTree'
import FileTreeRow from './FileTreeRow.vue'

const emit = defineEmits<{ open: [path: string] }>()

const { rows, errorMessage, tap, dropDownload } = useFileTree((path) => emit('open', path))
</script>

<template>
  <section>
    <p v-if="errorMessage" role="alert" data-testid="error">{{ errorMessage }}</p>
    <ul class="file-tree__list">
      <FileTreeRow
        v-for="row in rows"
        :key="row.path"
        :row="row"
        @tap="tap(row)"
        @drop-download="dropDownload(row)"
      />
    </ul>
  </section>
</template>

<style scoped>
.file-tree__list {
  margin: 0;
  padding: 0;
  list-style: none;
}
</style>
