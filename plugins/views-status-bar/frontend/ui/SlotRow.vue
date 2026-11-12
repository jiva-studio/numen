<script setup lang="ts">
/**
 * The entries of one slot of the bar, in order: a command is a button, a separator a gap, a view is
 * drawn in place, and a submenu or a command with items is a button that opens its menu.
 */
import type { Component } from 'vue'
import type { CommandItem, SlotItem, Caught } from '@numen/plugin-sdk'
import { formatTooltip } from '../lib/entries'
import SlotMenuButton from './SlotMenuButton.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  place: string
  entries: readonly SlotItem[]
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: Caught) => void) | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'run', command: string, itemId?: string, values?: Readonly<Record<string, string>>): void
}>()

/* -------------------------------- Handlers -------------------------------- */
function onRun(command: string, itemId?: string, values?: Readonly<Record<string, string>>): void {
  if (values) emit('run', command, itemId, values)
  else if (itemId === undefined) emit('run', command)
  else emit('run', command, itemId)
}
</script>

<template>
  <div class="status-bar__slot" :data-status-bar-slot="place">
    <template v-for="entry in entries" :key="entry.key">
      <span v-if="entry.kind === 'separator'" class="status-bar__gap" data-status-bar-gap />
      <SlotMenuButton
        v-else-if="entry.kind === 'command' && entry.hasItems"
        :entries="[entry]"
        :label="entry.title"
        :icon="entry.icon"
        :tooltip="formatTooltip(entry)"
        is-item-list
        :list-items="listItems"
        :report-error="reportError"
        @run="onRun"
      />
      <button
        v-else-if="entry.kind === 'command'"
        type="button"
        class="status-bar__entry"
        data-status-bar-entry
        :title="formatTooltip(entry)"
        :aria-label="entry.title"
        @click="onRun(entry.command, undefined, entry.values)"
      >
        <component :is="entry.icon as Component" v-if="entry.icon" :size="14" aria-hidden="true" />
        <template v-else>{{ entry.title }}</template>
      </button>
      <component :is="entry.component as Component" v-else-if="entry.kind === 'view'" />
      <SlotMenuButton
        v-else
        :entries="entry.entries"
        :label="entry.title"
        :icon="entry.icon"
        :list-items="listItems"
        :report-error="reportError"
        @run="onRun"
      />
    </template>
  </div>
</template>
