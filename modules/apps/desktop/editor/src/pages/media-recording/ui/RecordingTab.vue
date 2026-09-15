<script setup lang="ts">
/**
 * A recording tab: the controls its sound is played by at the top of the pane,
 * and under them the transcript.
 *
 * The sound itself plays where being drawn cannot reach it, so the controls
 * only say what a person did and the window plays it.
 */
import { computed } from 'vue'
import { Player } from '@numen/ui'
import { MediaLayout } from '@/entities/media'
import { DELETE_TEXT, PROOFREAD, WORDS as words } from '@/entities/media'
import type { MediaTabState } from '@/entities/media'

// --- Props & Emits ---
const props = defineProps<{ state: MediaTabState }>()

// --- State ---
const { deletable, now, playable, playing, proofreadable, runs } = props.state

/** What the menu offers over this recording: each item only where it applies. */
const offered = computed(() => [
  ...(proofreadable.value ? [{ id: PROOFREAD, text: words.proofread }] : []),
  ...(deletable.value ? [{ id: DELETE_TEXT, text: words.deleteText }] : []),
])

// --- Handlers ---
function onChoose(id: string) {
  if (id === PROOFREAD) props.state.proofread()
  if (id === DELETE_TEXT) props.state.deleteTranscript()
}

function onPlay() {
  props.state.play()
}

function onPause() {
  props.state.pause()
}

function onSeek(at: number) {
  props.state.go(at)
}

// --- Helpers ---
</script>

<template>
  <MediaLayout :state="props.state" :offered="offered" @choose="onChoose">
    <template #player>
      <Player
        v-if="playable"
        class="recording__player"
        :at="now"
        :length="runs"
        :playing="playing"
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
