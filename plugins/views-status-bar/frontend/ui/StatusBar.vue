<script setup lang="ts">
/**
 * The bar at the bottom of the window: the entries of `statusBar.left` at its start and those of
 * `statusBar.right` at its end.
 */
import type { SlotItem } from '@numen/plugin-sdk'
import { WORDS as words } from '../lib/entries'
import SlotRow from './SlotRow.vue'
import './statusBar.css'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  startEntries: readonly SlotItem[]
  endEntries: readonly SlotItem[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', command: string): void }>()
</script>

<template>
  <footer class="status-bar" role="region" :aria-label="words.bar">
    <SlotRow
      place="statusBar.left"
      :entries="startEntries"
      @run="(command: string) => emit('run', command)"
    />
    <SlotRow
      class="status-bar__slot--end"
      place="statusBar.right"
      :entries="endEntries"
      @run="(command: string) => emit('run', command)"
    />
  </footer>
</template>
