<script setup lang="ts">
/**
 * The status bar indicator of a new release, and the popover with its details.
 */
import { ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
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
  <div v-if="update" ref="root" class="update-indicator">
    <button
      type="button"
      class="update-indicator__trigger"
      :aria-expanded="isOpen"
      :aria-label="words.indicatorLabel"
      @click="onToggle"
    >
      <span class="update-indicator__label">v{{ update.version }}</span>
    </button>

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
