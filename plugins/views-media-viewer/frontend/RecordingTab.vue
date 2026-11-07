<script setup lang="ts">
/**
 * A recording tab: the picture of a video or the page that frames a player, the controls its sound
 * is played by, and under them the text: a transcript, captions or an article.
 *
 * Media plays through the window's single player, so the controls only report the user's actions.
 */
import { computed, onMounted, onUnmounted, watch } from 'vue'
import Player from './Player.vue'
import VideoSurface from './VideoSurface.vue'
import MediaLayout from './MediaLayout.vue'
import { useTranscript, type TranscriptOptions } from './model/transcript'
import type { ReaderTabState, Selector } from '@numen/plugin-sdk'
import type { Recordings, TextRange } from './types'
import { createTimeSelector, getSelectorStart } from './lib/place'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly path?: string
  readonly file?: string
  readonly state?: Partial<ReaderTabState> & { file?: string; [key: string]: unknown }
  /** What the tab reads the recording through, and how it plays it. The window's own by default. */
  readonly recordings?: Recordings
  readonly how?: TranscriptOptions
}>()

/* --------------------------------- State ---------------------------------- */
const targetPath = props.path || props.file || props.state?.path || props.state?.file || ''

const tab = useTranscript(props.recordings, targetPath, props.how)
const { now, runs, isPlaying, isUnplayable, isVideo, isLoaded, url } = tab.playback
const { pageUrl } = tab.content
/** Whether there is media to play: a note that names neither media nor a page draws no player. */
const hasMedia = computed(() => url.value !== '' || !targetPath.toLowerCase().endsWith('.md'))

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (props.state) {
    const shared = props.state as Record<string, unknown>
    shared.focusSpans = (...spans: readonly TextRange[]) => tab.navigation.reach(...spans)
  }
})

// The window sets a selector for a search hit and for the place saved before a restart.
watch(
  () => props.state?.selector,
  (selector: Selector | null | undefined) => {
    const start = getSelectorStart(selector)
    if (start !== null) tab.playback.controls.go(start)
  },
  { immediate: true },
)

/** Saves the current playback time as the place the tab opens at after a restart. */
const savePlace = () => props.state?.savePlace?.(createTimeSelector(now.value))

// Saved when playback stops and every ten seconds of it, so a restart loses at most that much.
watch(isPlaying, (playing) => {
  if (!playing) savePlace()
})
watch(
  () => Math.floor(now.value / 10_000),
  () => {
    if (isPlaying.value) savePlace()
  },
)

onUnmounted(() => {
  savePlace()
  tab.close()
})

/* -------------------------------- Handlers -------------------------------- */
function onPlay() {
  tab.playback.controls.play()
}

function onPause() {
  tab.playback.controls.pause()
}

function onSeek(ms: number) {
  tab.playback.controls.go(ms)
  props.state?.savePlace?.(createTimeSelector(ms))
}
</script>

<template>
  <MediaLayout :state="tab">
    <template #picture>
      <iframe
        v-if="pageUrl"
        class="recording__page"
        :src="pageUrl"
        :title="words.page"
        allow="fullscreen; picture-in-picture; encrypted-media"
        sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"
      />
      <VideoSurface v-else-if="isVideo" :player="tab.playback.player" :is-shown="isLoaded" />
    </template>
    <template v-if="!pageUrl && hasMedia" #player>
      <Player
        v-if="!isUnplayable"
        class="recording__player"
        :at="now"
        :length="runs"
        :is-playing="isPlaying"
        :label="words.player"
        @play="onPlay"
        @pause="onPause"
        @seek="onSeek"
      />
      <p v-else class="recording__note">{{ words.unplayable }}</p>
    </template>
  </MediaLayout>
</template>

<style scoped>
.recording__player {
  flex: 1;
  min-inline-size: 0;
}

.recording__page {
  flex: none;
  inline-size: 100%;
  aspect-ratio: 16 / 9;
  max-block-size: 24rem;
  border: 0;
}

.recording__note {
  flex: none;
  margin: 0;
  color: var(--numen-hushed);
}
</style>
