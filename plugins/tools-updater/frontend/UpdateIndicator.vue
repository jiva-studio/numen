<script setup lang="ts">
/**
 * The title bar indicator of a new release, and the popover with its details.
 */
import { computed, ref, watch } from 'vue'
import { BarChip } from '@numen/ui'
import type { Update } from './api/updater'
import { getTooltip } from './model/tooltip'
import type { Platform } from './types'
import UpdatePopover from './UpdatePopover.vue'
import { WORDS as words } from './words'

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

/* -------------------------------- Watchers -------------------------------- */
// A later announcement starts closed, not open as the last one was left.
watch(
  () => props.update,
  (update) => {
    if (update === null) isOpen.value = false
  },
)

/* -------------------------------- Handlers -------------------------------- */
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
  <BarChip
    v-if="update"
    v-model:open="isOpen"
    tone="accent"
    :label="tooltip"
    :tooltip="tooltip"
    :popover-label="words.title"
  >
    v{{ update.version }}
    <template #popover>
      <UpdatePopover
        :update="update"
        :platform="platform"
        @close="onClose"
        @download="onDownload"
        @release-notes="onReleaseNotes"
        @dismiss="onDismiss"
      />
    </template>
  </BarChip>
</template>
