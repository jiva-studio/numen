<script setup lang="ts">
/**
 * A recording tab: the controls its sound is played by at the top of the pane, and under them the
 * transcript.
 *
 * The sound plays through the window's one player, so the controls only say what a person did.
 */
import { computed, onMounted, onUnmounted, watch } from 'vue'
import Player from './Player.vue'
import MediaLayout from './MediaLayout.vue'
import { recordings as servedRecordings } from './api/wire'
import { useTranscript, type TranscriptOptions } from './model/transcript'
import type { Recordings, Span } from './types'
import { DELETE_TEXT, PROOFREAD, WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly path?: string
  readonly file?: string
  readonly state?: { path?: string; file?: string; spans?: readonly Span[]; [key: string]: unknown }
  /** What the tab reads the recording through, and how it plays it. The window's own by default. */
  readonly recordings?: Recordings
  readonly how?: TranscriptOptions
}>()

/* --------------------------------- State ---------------------------------- */
const targetPath = props.path || props.file || props.state?.path || props.state?.file || ''

const tab = useTranscript(props.recordings ?? servedRecordings, targetPath, props.how)
const { now, runs, isPlaying, isUnplayable } = tab.playback
const { isProofreadable, isDeletable } = tab.offers

/** What the menu offers over this recording: each item only where it applies. */
const menuItems = computed(() => [
  ...(isProofreadable.value ? [{ id: PROOFREAD, text: words.proofread }] : []),
  ...(isDeletable.value ? [{ id: DELETE_TEXT, text: words.deleteText }] : []),
])

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  if (props.state) {
    const shared = props.state as Record<string, unknown>
    shared.focusSpans = (...spans: readonly Span[]) => tab.navigation.reach(...spans)
  }
  const spans = props.state?.spans
  if (spans && spans.length > 0) void tab.navigation.reach(...spans)
})

watch(
  () => props.state?.spans,
  (spans) => {
    if (spans && spans.length > 0) void tab.navigation.reach(...spans)
  },
)

onUnmounted(() => tab.close())

/* -------------------------------- Handlers -------------------------------- */
function onChoose(id: string) {
  if (id === PROOFREAD) void tab.offers.proofread()
  if (id === DELETE_TEXT) void tab.offers.deleteTranscript()
}

function onPlay() {
  tab.playback.controls.play()
}

function onPause() {
  tab.playback.controls.pause()
}

function onSeek(ms: number) {
  tab.playback.controls.go(ms)
}
</script>

<template>
  <MediaLayout :state="tab" :menu-items="menuItems" @choose="onChoose">
    <template #player>
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

.recording__note {
  flex: none;
  margin: 0;
  color: var(--numen-hushed);
}
</style>
