<script setup lang="ts">
/**
 * The title bar indicator of a new release, and the popover with its details.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { isPageMove, useMenuDismissal } from '@numen/ui'
import type { Update } from '../api/updater'
import { getTooltip } from '../model/tooltip'
import type { Platform } from '../types'
import UpdatePopover from './UpdatePopover.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  version: string
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
const tooltip = computed(() => (props.update ? getTooltip(props.version, props.update) : ''))
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)

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
      tabindex="-1"
      class="update-indicator__trigger"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :aria-label="tooltip"
      :title="tooltip"
      @click="onToggle"
    >
      <span class="update-indicator__label">v{{ update.version }}</span>
    </button>

    <UpdatePopover
      v-if="isOpen && update"
      :update="update"
      :platform="platform"
      @close="onClose"
      @download="onDownload"
      @release-notes="onReleaseNotes"
      @dismiss="onDismiss"
    />
  </div>
</template>
