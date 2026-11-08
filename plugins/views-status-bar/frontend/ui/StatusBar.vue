<script setup lang="ts">
/**
 * The bar at the bottom of the window: the entries of `statusBar.left` at its start and those of
 * `statusBar.right` at its end.
 */
import type { CommandItem, SlotItem } from '@numen/plugin-sdk'
import { WORDS as words } from '../lib/entries'
import SlotRow from './SlotRow.vue'
import './statusBar.css'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  startEntries: readonly SlotItem[]
  endEntries: readonly SlotItem[]
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: (error: unknown) => void
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', command: string, argument?: string): void }>()

/* -------------------------------- Handlers -------------------------------- */
function onRun(command: string, argument?: string): void {
  if (argument === undefined) emit('run', command)
  else emit('run', command, argument)
}
</script>

<template>
  <footer class="status-bar" role="region" :aria-label="words.bar">
    <SlotRow
      place="statusBar.left"
      :entries="startEntries"
      :list-items="listItems"
      :report-error="reportError"
      @run="onRun"
    />
    <SlotRow
      class="status-bar__slot--end"
      place="statusBar.right"
      :entries="endEntries"
      :list-items="listItems"
      :report-error="reportError"
      @run="onRun"
    />
  </footer>
</template>
