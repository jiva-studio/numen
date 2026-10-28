<script setup lang="ts">
/**
 * Status bar indicator and details popover announcing new application releases.
 */
import { onUnmounted, ref, useTemplateRef, watch } from 'vue'
import { listenWindowPointerAndEscape } from '@/shared/lib/dom'
import { useAppUpdate, type UpdateOptions } from '../model/useAppUpdate'
import UpdatePopover from './UpdatePopover.vue'
import './updateIndicator.css'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    options?: UpdateOptions
  }>(),
  {
    options: () => ({}),
  },
)

/* --------------------------------- State ---------------------------------- */
const {
  hasUpdate,
  isDismissed,
  updateInfo,
  platform,
  dismissUpdate,
  openDownload,
  openReleaseNotes,
} = useAppUpdate(props.options)

const isOpen = ref(false)
const rootRef = useTemplateRef<HTMLElement>('rootRef')

/* --------------------------------- Hooks ---------------------------------- */
let releaseListeners: (() => void) | null = null

watch(isOpen, (open) => {
  releaseListeners?.()
  releaseListeners = open ? listenWindowPointerAndEscape(onPointerDown, onKeyDown) : null
})

onUnmounted(() => {
  releaseListeners?.()
  releaseListeners = null
})

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}

function onDownload(press?: MouseEvent) {
  if (!press?.defaultPrevented) {
    openDownload()
  }
  onClose()
}

function onReleaseNotes(press?: MouseEvent) {
  if (!press?.defaultPrevented) {
    openReleaseNotes()
  }
  onClose()
}

function onDismiss() {
  dismissUpdate()
  onClose()
}

function onPointerDown(event: PointerEvent) {
  const el = rootRef.value
  if (!el || !(event.target instanceof Node)) {
    return
  }
  if (!el.contains(event.target)) {
    onClose()
  }
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    onClose()
  }
}
</script>

<template>
  <div v-if="hasUpdate && !isDismissed && updateInfo" ref="rootRef" class="update-indicator">
    <button
      type="button"
      class="update-indicator__trigger"
      :aria-expanded="isOpen"
      aria-label="Application update available"
      @click="onToggle"
    >
      <span class="update-indicator__label">v{{ updateInfo.latestVersion }}</span>
    </button>

    <UpdatePopover
      v-if="isOpen"
      :update-info="updateInfo"
      :platform="platform"
      @close="onClose"
      @download="onDownload"
      @release-notes="onReleaseNotes"
      @dismiss="onDismiss"
    />
  </div>
</template>
