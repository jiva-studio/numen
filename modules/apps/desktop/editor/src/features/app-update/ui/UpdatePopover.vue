<script setup lang="ts">
/**
 * Interactive popover detailing new release information and download links.
 */
import { computed } from 'vue'
import type { UpdateInfo } from '../model/types'
import { formatReleaseDate, type ClientPlatform } from '../lib/release'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  updateInfo: UpdateInfo
  platform?: ClientPlatform | undefined
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'download'): void
  (event: 'release-notes'): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const formattedDate = computed(() => {
  return formatReleaseDate(props.updateInfo.releasedAt)
})

const platformMessage = computed(() => {
  if (props.platform === 'windows') {
    return 'Download the installer to update Numen.'
  }
  if (props.platform === 'darwin') {
    return 'Download the disk image to update Numen.'
  }
  return 'View release notes and packages on GitHub.'
})

const isSingleAction = computed(() => {
  return (
    props.platform === 'linux' ||
    !props.updateInfo.notesUrl ||
    props.updateInfo.downloadUrl === props.updateInfo.notesUrl
  )
})

const actionLabel = computed(() => {
  if (props.platform === 'linux' || props.updateInfo.downloadUrl.includes('github.com')) {
    return 'View on GitHub'
  }
  return 'Download update'
})

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  emit('close')
}

function onDownload() {
  emit('download')
}

function onReleaseNotes() {
  emit('release-notes')
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
        {{ actionLabel }}
      </a>

      <a
        v-if="!isSingleAction && updateInfo.notesUrl"
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
