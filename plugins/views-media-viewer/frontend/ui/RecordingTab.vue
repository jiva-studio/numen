<script setup lang="ts">
/**
 * A recording tab: the picture of a video or the page that frames a player, the controls its sound
 * is played by, and under them the text: a transcript, captions or an article.
 *
 * Media plays through the window's single player, so the controls only report the user's actions.
 */
import { computed } from 'vue'
import Player from './Player.vue'
import VideoSurface from './VideoSurface.vue'
import MediaLayout from './MediaLayout.vue'
import { useTranscript, type TranscriptOptions } from '../model/transcript'
import { useRecordingPlace } from '../model/useRecordingPlace'
import { isNotePath, type ReaderTabState } from '@numen/plugin-sdk'
import type { Recordings } from '../types'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly path?: string
  readonly file?: string
  readonly state?: Partial<ReaderTabState>
  readonly recordings?: Recordings
  readonly how?: TranscriptOptions
}>()

/* --------------------------------- State ---------------------------------- */
const targetPath = props.path || props.file || props.state?.path || ''

const tab = useTranscript(props.recordings, targetPath, props.how)
const { source, progress } = tab.playback
const { isUnplayable, isVideo, isLoaded, url } = source
const { now, runs, isPlaying } = progress
const { pageUrl } = tab.content
/** Whether there is media to play: a note that names neither media nor a page draws no player. */
const hasMedia = computed(() => url.value !== '' || !isNotePath(targetPath))
const place = useRecordingPlace(tab, () => props.state)

/* -------------------------------- Handlers -------------------------------- */
function onPlay() {
  tab.playback.controls.play()
}

function onPause() {
  tab.playback.controls.pause()
}

function onSeek(ms: number) {
  place.seekTo(ms)
}
</script>

<template>
  <MediaLayout :state="tab">
    <template #picture>
      <!-- The page frames one player, which inherits these flags; without same-origin the player gets no Referer and refuses to play. -->
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
  --page-max-block-size: 24rem;
  flex: none;
  inline-size: 100%;
  aspect-ratio: 16 / 9;
  max-block-size: var(--page-max-block-size);
  border: 0;
}

.recording__note {
  flex: none;
  margin: 0;
  color: var(--numen-hushed);
}
</style>
