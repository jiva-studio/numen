<script setup lang="ts">
/**
 * The files of the vault, held and not downloaded. Tapping a file that is not downloaded downloads it and
 * opens it; a downloaded file opens at once. The page emits `open` with the path of the file to open.
 */
defineEmits<{ open: [path: string] }>()
</script>

<template>
  <div />
</template>
