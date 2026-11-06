<script setup lang="ts">
/**
 * A submenu of the bar: a button with its title, and the list of its entries it opens above the
 * bar. Running a command of the list closes it, and so do a press outside it and Escape.
 */
import { ref, useTemplateRef, watch, type Component } from 'vue'
import { ChevronUp } from '@lucide/vue'
import { useMenuDismissal } from '@numen/ui'
import type { SlotSubmenu } from '@numen/plugin-sdk'
import SlotMenu from './SlotMenu.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  submenu: SlotSubmenu
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{ (event: 'run', command: string): void }>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose)

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => (open ? dismissal.listen() : dismissal.release()))

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}

function onRun(command: string) {
  isOpen.value = false
  emit('run', command)
}
</script>

<template>
  <div ref="root" class="status-bar__submenu">
    <button
      type="button"
      class="status-bar__entry"
      data-status-bar-entry
      aria-haspopup="menu"
      :aria-expanded="isOpen"
      @click="onToggle"
    >
      <component
        :is="submenu.icon as Component"
        v-if="submenu.icon"
        :size="14"
        aria-hidden="true"
      />
      {{ submenu.title }}
      <ChevronUp :size="12" aria-hidden="true" />
    </button>
    <SlotMenu
      v-if="isOpen"
      class="status-bar__dropdown"
      :title="submenu.title"
      :entries="submenu.entries"
      @run="onRun"
    />
  </div>
</template>
