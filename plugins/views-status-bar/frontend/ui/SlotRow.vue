<script setup lang="ts">
/**
 * The entries of one slot of the bar, in order: a command is a button, a separator a gap, a view is
 * drawn in place, and a submenu or a command with items is a button that opens its menu.
 */
import type { Component } from 'vue'
import {
  SLOT_ATTRIBUTE,
  SLOT_ENTRY_ATTRIBUTE,
  type CommandItem,
  type SlotItem,
} from '@numen/plugin-sdk'
import { formatTooltip } from '../lib/entries'
import SlotMenuButton from './SlotMenuButton.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  place: string
  entries: readonly SlotItem[]
  listItems: (command: string) => Promise<readonly CommandItem[]>
  reportError?: ((error: unknown) => void) | undefined
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

/* -------------------------------- Helpers --------------------------------- */
/** The attribute the window finds a command or a view by when it drags the entries of the bars. */
function markEntry(entry: SlotItem): Record<string, string> {
  return { [SLOT_ENTRY_ATTRIBUTE]: entry.key }
}
</script>

<template>
  <div
    class="status-bar__slot"
    :class="{ 'status-bar__slot--empty': entries.length === 0 }"
    :data-status-bar-slot="place"
    v-bind="{ [SLOT_ATTRIBUTE]: place }"
  >
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
        v-bind="markEntry(entry)"
        @run="onRun"
      />
      <button
        v-else-if="entry.kind === 'command'"
        type="button"
        class="status-bar__entry"
        data-status-bar-entry
        v-bind="markEntry(entry)"
        :title="formatTooltip(entry)"
        :aria-label="entry.title"
        @click="onRun(entry.command, undefined, entry.values)"
      >
        <component :is="entry.icon as Component" v-if="entry.icon" :size="14" aria-hidden="true" />
        <template v-else>{{ entry.title }}</template>
      </button>
      <div v-else-if="entry.kind === 'view'" class="status-bar__view" v-bind="markEntry(entry)">
        <component :is="entry.component as Component" />
      </div>
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
