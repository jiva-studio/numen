<script setup lang="ts">
/** A note: its Markdown set as prose. */
import { Prose } from '@numen/ui'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ body: string }>()
</script>

<template>
  <div class="note">
    <Prose :text="body" />
  </div>
</template>

<style scoped>
.note {
  block-size: 100%;
  overflow-y: auto;
  padding: 0.75rem var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-reading-size);
  color: var(--numen-ink);
  user-select: text;
  -webkit-user-select: text;
}
</style>
