<script setup lang="ts">
/**
 * Interactive popover detailing new release information and download links.
 */
import { computed } from 'vue'
import type { UpdateInfo } from '../model/types'
import {
  formatReleaseDate,
  getPlatformUpdateMessage,
  type ClientPlatform,
} from '../lib/release'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  updateInfo: UpdateInfo
  platform?: ClientPlatform | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'download', press: MouseEvent): void
  (event: 'release-notes', press: MouseEvent): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const formattedDate = computed(() => {
  return formatReleaseDate(props.updateInfo.releasedAt)
})

const platformMessage = computed(() => getPlatformUpdateMessage(props.platform))

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}

function onDownload(press: MouseEvent) {
  emit('download', press)
}

function onReleaseNotes(press: MouseEvent) {
  emit('release-notes', press)
}

function onDismiss() {
  emit('dismiss')
}
</script>

<template>
  <div class="update-popover" role="dialog" aria-modal="false">
    <header class="update-popover__header">
      <div>
        <span class="update-popover__title">Update available</span>
        <span class="update-popover__badge">v{{ updateInfo.latestVersion }}</span>
      </div>
      <button
        type="button"
        class="update-popover__close-btn"
        aria-label="Close update popover"
        @click="onClose"
      >
        ✕
      </button>
    </header>

    <div class="update-popover__body">
      <p class="update-popover__desc">{{ platformMessage }}</p>
      <p v-if="formattedDate" class="update-popover__date">Released on {{ formattedDate }}</p>
    </div>

    <footer class="update-popover__actions">
      <a
        :href="updateInfo.downloadUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="update-popover__btn update-popover__btn--primary"
        @click="onDownload"
      >
        Download
      </a>

      <a
        v-if="updateInfo.notesUrl && updateInfo.notesUrl !== updateInfo.downloadUrl"
        :href="updateInfo.notesUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="update-popover__btn update-popover__btn--ghost"
        @click="onReleaseNotes"
      >
        Release notes
      </a>

      <button
        type="button"
        class="update-popover__btn update-popover__btn--quiet"
        @click="onDismiss"
      >
        Dismiss
      </button>
    </footer>
  </div>
</template>
