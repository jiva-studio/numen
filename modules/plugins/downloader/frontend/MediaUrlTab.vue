<script setup lang="ts">
/**
 * Media URL downloader tab view.
 *
 * Downloads media from web addresses and renders an embed player with cues.
 */
import { computed, ref, useId, useTemplateRef } from 'vue'
import type { Cue } from '@numen/protocol'
import { downloaderClient } from './client'
import MediaEmbed from './embed/MediaEmbed.vue'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
interface CueItem {
  readonly startMs: number
  readonly endMs: number
  readonly text: string
}

const props = defineProps<{
  readonly state?: unknown
  readonly at?: string
  readonly path?: string
}>()

/* --------------------------------- State ---------------------------------- */
const initialUrl = computed(() => {
  const saved = props.state as
    { url?: { value?: string } | string; path?: string; at?: string } | undefined
  if (typeof saved?.url === 'string') return saved.url
  if (typeof saved?.url?.value === 'string') return saved.url.value
  return props.path || props.at || ''
})

const inputUrl = ref(initialUrl.value)
const activeUrl = ref(initialUrl.value)
const isDownloading = ref(false)
const errorMessage = ref('')
const statusMessage = ref('')
const cues = ref<CueItem[]>([])
const mediaTitle = ref('')

const urlFieldId = useId()

const player = useTemplateRef<{ seek(ms: number): void }>('player')

const hasPlayer = computed(() => Boolean(activeUrl.value))
const hasCues = computed(() => cues.value.length > 0)

/* -------------------------------- Handlers -------------------------------- */
async function onDownload(): Promise<void> {
  const target = inputUrl.value.trim()
  if (!target) return

  isDownloading.value = true
  errorMessage.value = ''
  statusMessage.value = words.checking

  try {
    const handleCheck = await downloaderClient.canHandle({ url: target })
    if (!handleCheck.canHandle) {
      errorMessage.value = words.unsupportedUrl
      statusMessage.value = ''
      isDownloading.value = false
      return
    }

    statusMessage.value = words.downloading
    const textResponse = await downloaderClient.getText({ url: target })
    if (textResponse.title) {
      mediaTitle.value = textResponse.title
    }
    if (textResponse.cues && textResponse.cues.length > 0) {
      cues.value = textResponse.cues.map((c: Cue) => ({
        startMs: c.from,
        endMs: c.to,
        text: c.text,
      }))
    }

    activeUrl.value = target
    statusMessage.value = words.success
  } catch (err) {
    errorMessage.value = err instanceof Error ? err.message : words.error
    statusMessage.value = ''
  } finally {
    isDownloading.value = false
  }
}

function onSeekCue(ms: number): void {
  player.value?.seek(ms)
}

function onTimeUpdate(_ms: number): void {
  // Seek sync hook
}

/* -------------------------------- Helpers --------------------------------- */
function formatTime(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}
</script>

<template>
  <div class="media-downloader">
    <header class="media-downloader__toolbar">
      <form class="media-downloader__form" @submit.prevent="onDownload">
        <label :for="urlFieldId" class="sr-only">{{ words.mediaUrl }}</label>
        <input
          :id="urlFieldId"
          v-model="inputUrl"
          type="url"
          class="media-downloader__input"
          :placeholder="words.enterUrl"
          :disabled="isDownloading"
        />
        <button
          type="submit"
          class="media-downloader__button"
          :disabled="isDownloading || !inputUrl.trim()"
        >
          {{ words.download }}
        </button>
      </form>

      <div v-if="statusMessage" class="media-downloader__status">
        {{ statusMessage }}
      </div>
      <div v-if="errorMessage" class="media-downloader__error">
        {{ errorMessage }}
      </div>
    </header>

    <main class="media-downloader__content">
      <h2 v-if="mediaTitle" class="media-downloader__title">
        {{ mediaTitle }}
      </h2>

      <div v-if="hasPlayer" class="media-downloader__player-wrapper">
        <MediaEmbed
          ref="player"
          :embed="activeUrl"
          :copy="''"
          :words="words"
          @time-update="onTimeUpdate"
        />
      </div>

      <section v-if="hasCues" class="media-downloader__transcript">
        <h3 class="media-downloader__subtitle">{{ words.transcript }}</h3>
        <ul class="media-downloader__cue-list">
          <li
            v-for="(cue, index) in cues"
            :key="index"
            class="media-downloader__cue"
            @click="onSeekCue(cue.startMs)"
          >
            <span class="media-downloader__cue-time">{{ formatTime(cue.startMs) }}</span>
            <span class="media-downloader__cue-text">{{ cue.text }}</span>
          </li>
        </ul>
      </section>
    </main>
  </div>
</template>

<style scoped>
.media-downloader {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  inline-size: 100%;
  overflow-y: auto;
  padding: 1rem;
  box-sizing: border-box;
}

.media-downloader__toolbar {
  margin-bottom: 1rem;
}

.media-downloader__form {
  display: flex;
  gap: 0.5rem;
}

.media-downloader__input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  font-size: 0.9rem;
  border: 1px solid var(--numen-rule, #ccc);
  border-radius: 4px;
  background: var(--numen-field, #fff);
  color: var(--numen-ink, #000);
}

.media-downloader__button {
  padding: 0.5rem 1rem;
  font-size: 0.9rem;
  border: 1px solid transparent;
  border-radius: 4px;
  background: var(--numen-accent, #3b82f6);
  color: #fff;
  cursor: pointer;
}

.media-downloader__button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.media-downloader__status {
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: var(--numen-muted, #666);
}

.media-downloader__error {
  margin-top: 0.5rem;
  font-size: 0.85rem;
  color: var(--numen-error, #ef4444);
}

.media-downloader__content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.media-downloader__title {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.media-downloader__player-wrapper {
  inline-size: 100%;
}

.media-downloader__transcript {
  margin-top: 1rem;
}

.media-downloader__subtitle {
  margin: 0 0 0.5rem 0;
  font-size: 1rem;
  font-weight: 600;
}

.media-downloader__cue-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.media-downloader__cue {
  display: flex;
  gap: 0.75rem;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  cursor: pointer;
}

.media-downloader__cue:hover {
  background: var(--numen-hover, rgba(0, 0, 0, 0.05));
}

.media-downloader__cue-time {
  font-family: monospace;
  font-size: 0.85rem;
  color: var(--numen-accent, #3b82f6);
}

.media-downloader__cue-text {
  font-size: 0.9rem;
}
</style>
