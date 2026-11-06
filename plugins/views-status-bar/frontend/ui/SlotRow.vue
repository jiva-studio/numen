<script setup lang="ts">
/**
 * The entries of one slot of the bar, in order: a command is a button, a separator a gap, a view is
 * drawn in place, and a submenu is a button that opens its list.
 */
import type { Component } from 'vue'
import type { SlotItem } from '@numen/plugin-sdk'
import { formatTooltip } from '../lib/entries'
import SlotSubmenu from './SlotSubmenu.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  place: string
  entries: readonly SlotItem[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', command: string): void }>()
</script>

<template>
  <div class="status-bar__slot" :data-status-bar-slot="place">
    <template v-for="entry in entries" :key="entry.key">
      <span v-if="entry.kind === 'separator'" class="status-bar__gap" data-status-bar-gap />
      <button
        v-else-if="entry.kind === 'command'"
        type="button"
        class="status-bar__entry"
        data-status-bar-entry
        :title="formatTooltip(entry)"
        :aria-label="entry.title"
        @click="emit('run', entry.command)"
      >
        <component :is="entry.icon as Component" v-if="entry.icon" :size="14" aria-hidden="true" />
        <template v-else>{{ entry.title }}</template>
      </button>
      <component :is="entry.component as Component" v-else-if="entry.kind === 'view'" />
      <SlotSubmenu v-else :submenu="entry" @run="(command: string) => emit('run', command)" />
    </template>
  </div>
</template>
