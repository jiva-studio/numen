<script setup lang="ts">
/**
 * The title bar indicator of a new release, and the popover with its details.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { State, Update } from './api/updater'
import { getTooltip } from './lib/tooltip'
import type { Platform } from './types'
import UpdatePopover from './UpdatePopover.vue'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  version: string
  state: State
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
const tooltip = computed(() => getTooltip(props.version, props.update, props.state))
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
  <div
    ref="root"
    class="update-indicator"
    :class="{ 'update-indicator--available': update !== null }"
  >
    <button
      v-if="update"
      type="button"
      class="update-indicator__trigger"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :aria-label="tooltip"
      :title="tooltip"
      @click="onToggle"
    >
      <span class="update-indicator__label">v{{ update.version }}</span>
    </button>
    <span v-else class="update-indicator__trigger" role="img" :aria-label="tooltip" :title="tooltip">
      <span class="update-indicator__label">v{{ version }}</span>
    </span>

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
