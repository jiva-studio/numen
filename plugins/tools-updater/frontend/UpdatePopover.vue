<script setup lang="ts">
/**
 * The details of a new release and the actions on it.
 */
import { computed } from 'vue'
import type { Update } from './api/updater'
import type { Platform } from './types'
import { formatReleaseDate, getPlatformMessage } from './lib/release'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  update: Update
  platform: Platform
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'download'): void
  (event: 'release-notes'): void
  (event: 'dismiss'): void
}>()

/* --------------------------------- State ---------------------------------- */
const releaseDate = computed(() => formatReleaseDate(props.update.releasedAt))
const platformMessage = computed(() => getPlatformMessage(props.platform))
const hasNotes = computed(
  () => props.update.notesUrl !== '' && props.update.notesUrl !== props.update.downloadUrl,
)

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
  <div class="update-popover" role="dialog" aria-modal="false" :aria-label="words.title">
    <header class="update-popover__header">
      <div>
        <span class="update-popover__title">{{ words.title }}</span>
        <span class="update-popover__badge">v{{ update.version }}</span>
      </div>
      <button
        type="button"
        class="update-popover__close-btn"
        :aria-label="words.close"
        @click="onClose"
      >
        ✕
      </button>
    </header>

    <div class="update-popover__body">
      <p class="update-popover__desc">{{ platformMessage }}</p>
      <p v-if="releaseDate" class="update-popover__date">
        {{ words.releasedOn }} {{ releaseDate }}
      </p>
    </div>

    <footer class="update-popover__actions">
      <button
        type="button"
        class="update-popover__btn update-popover__btn--primary"
        @click="onDownload"
      >
        {{ words.download }}
      </button>

      <button
        v-if="hasNotes"
        type="button"
        class="update-popover__btn update-popover__btn--ghost"
        @click="onReleaseNotes"
      >
        {{ words.releaseNotes }}
      </button>

      <button
        type="button"
        class="update-popover__btn update-popover__btn--quiet"
        @click="onDismiss"
      >
        {{ words.dismiss }}
      </button>
    </footer>
  </div>
</template>
