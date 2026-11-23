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
    <ul>
      <li v-for="row in rows" :key="row.path" data-testid="file-row" @click="tap(row)">
        <span>{{ row.name }}</span>
        <span v-if="row.isDownloading"> Downloading</span>
        <span v-else-if="row.isAbsent"> Not downloaded</span>
        <button
          v-if="!row.isFolder && !row.isAbsent && !row.isDownloading"
          type="button"
          data-testid="drop-download"
          @click.stop="dropDownload(row)"
        >
          Remove download
        </button>
      </li>
    </ul>
  </section>
</template>
