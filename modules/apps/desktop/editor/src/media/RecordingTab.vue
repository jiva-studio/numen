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
import MediaTab from './MediaTab.vue'
import { DELETE_TEXT, PROOFREAD, WORDS as words } from './words'
import type { MediaTabState } from './kind'

const props = defineProps<{ state: MediaTabState }>()

const { deletable, now, playable, playing, proofreadable, runs } = props.state

/** What the menu offers over this recording: each item only where it applies. */
const offered = computed(() => [
  ...(proofreadable.value ? [{ id: PROOFREAD, text: words.proofread }] : []),
  ...(deletable.value ? [{ id: DELETE_TEXT, text: words.deleteText }] : []),
])

const chose = (id: string) => {
  if (id === PROOFREAD) props.state.proofreads()
  if (id === DELETE_TEXT) props.state.deletes()
}
</script>

<template>
  <MediaTab :state="props.state" :offered="offered" @choose="chose">
    <template #player>
      <Player
        v-if="playable"
        class="recording__player"
        :at="now"
        :length="runs"
        :playing="playing"
        :label="words.player"
        @play="props.state.play()"
        @pause="props.state.pause()"
        @seek="props.state.go($event)"
      />
      <p v-else class="recording__note">{{ words.unplayable }}</p>
    </template>
  </MediaTab>
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
