<script setup lang="ts">
/**
 * The details of a new release and the actions on it.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { Update } from '../api/updater'
import type { Platform } from '../types'
import { formatReleaseDate, getPlatformMessage } from '../lib/release'
import { WORDS as words } from '../words'

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
      <span class="update-popover__title">{{ words.title }}</span>
      <Button
        variant="ghost"
        size="icon-small"
        class="update-popover__close-btn"
        :title="words.close"
        :aria-label="words.close"
        @click="onClose"
      >
        <svg viewBox="0 0 12 12" width="12" height="12" aria-hidden="true">
          <path d="M2 2l8 8M10 2l-8 8" stroke="currentColor" stroke-width="1.5" fill="none" />
        </svg>
      </Button>
    </header>

    <div class="update-popover__body">
      <p class="update-popover__desc">{{ platformMessage }}</p>
      <p class="update-popover__release">
        <span class="update-popover__version">v{{ update.version }}</span>
        <template v-if="releaseDate">{{ ` · ${words.releasedOn} ${releaseDate}` }}</template>
      </p>
    </div>

    <footer class="update-popover__actions">
      <Button size="small" @click="onDownload">{{ words.download }}</Button>
      <Button v-if="hasNotes" variant="outline" size="small" @click="onReleaseNotes">
        {{ words.releaseNotes }}
      </Button>
      <Button variant="ghost" size="small" class="update-popover__dismiss-btn" @click="onDismiss">
        {{ words.dismiss }}
      </Button>
    </footer>
  </div>
</template>
