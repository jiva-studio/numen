<script setup lang="ts">
/**
 * The status bar indicator of a new release, and the popover with its details.
 */
import { ref } from 'vue'
import type { Update } from './api/updater'
import type { Platform } from './types'
import UpdatePopover from './UpdatePopover.vue'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  update: Update | null
  platform: Platform
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'download'): void
  (event: 'release-notes'): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}

function onDownload() {
  emit('download')
  onClose()
}

function onReleaseNotes() {
  emit('release-notes')
  onClose()
}

function onDismiss() {
  emit('dismiss')
  onClose()
}
</script>

<template>
  <div v-if="update" class="update-indicator" @keydown.esc="onClose">
    <button
      type="button"
      class="update-indicator__trigger"
      :aria-expanded="isOpen"
      :aria-label="words.indicatorLabel"
      @click="onToggle"
    >
      <span class="update-indicator__label">v{{ update.version }}</span>
    </button>

    <div v-if="isOpen" class="update-indicator__backdrop" @click="onClose" />
    <UpdatePopover
      v-if="isOpen"
      :update="update"
      :platform="platform"
      @close="onClose"
      @download="onDownload"
      @release-notes="onReleaseNotes"
      @dismiss="onDismiss"
    />
  </div>
</template>
