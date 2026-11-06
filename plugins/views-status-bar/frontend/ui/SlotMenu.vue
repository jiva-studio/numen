<script setup lang="ts">
/**
 * The entries of a submenu as a list: a command is a row, a separator a line, a view a row that
 * holds it, and a submenu a row that opens its own list beside this one.
 */
import { ref, type Component } from 'vue'
import { ChevronRight } from '@lucide/vue'
import type { SlotItem } from '@numen/plugin-sdk'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  title: string
  entries: readonly SlotItem[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', command: string): void }>()

/* --------------------------------- State ---------------------------------- */
const openKey = ref<string | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(key: string) {
  openKey.value = openKey.value === key ? null : key
}
</script>

<template>
  <div class="status-bar-menu" role="menu" :aria-label="title">
    <template v-for="entry in entries" :key="entry.key">
      <div v-if="entry.kind === 'separator'" class="status-bar-menu__line" role="separator" />
      <button
        v-else-if="entry.kind === 'command'"
        type="button"
        class="status-bar-menu__row"
        role="menuitem"
        @click="emit('run', entry.command)"
      >
        <component :is="entry.icon as Component" v-if="entry.icon" :size="14" aria-hidden="true" />
        {{ entry.title }}
      </button>
      <div v-else-if="entry.kind === 'view'" class="status-bar-menu__view" role="none">
        <component :is="entry.component as Component" />
      </div>
      <template v-else>
        <button
          type="button"
          class="status-bar-menu__row"
          role="menuitem"
          aria-haspopup="menu"
          :aria-expanded="openKey === entry.key"
          @click="onToggle(entry.key)"
        >
          {{ entry.title }}
          <ChevronRight class="status-bar-menu__chevron" :size="12" aria-hidden="true" />
        </button>
        <SlotMenu
          v-if="openKey === entry.key"
          class="status-bar-menu--nested"
          :title="entry.title"
          :entries="entry.entries"
          @run="(command: string) => emit('run', command)"
        />
      </template>
    </template>
  </div>
</template>
