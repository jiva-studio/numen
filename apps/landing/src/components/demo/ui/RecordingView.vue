<script setup lang="ts">
/**
 * A recording: a player over the transcript. The transcript follows the playback while it plays.
 */
import { computed, ref } from 'vue'
import { LocateFixed, Pause, Play } from '@lucide/vue'
import { Slider, clock } from '@numen/ui'
import { LECTURE_CUES, LECTURE_MEDIA_URL, getActiveCueIndex } from '../lib/cues'
import type { Player } from '../model/player'
import { usePlayerLoad } from '../model/usePlayerLoad'
import TranscriptView from './TranscriptView.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ player: Player }>()

/* --------------------------------- State ---------------------------------- */
const isFollowing = ref(true)
const active = computed(() => getActiveCueIndex(LECTURE_CUES, props.player.position.value))

/* --------------------------------- Hooks ---------------------------------- */
usePlayerLoad(() => props.player, LECTURE_MEDIA_URL)
</script>

<template>
  <div class="recording">
    <div class="recording__bar">
      <button type="button" class="recording__action" aria-label="Play" @click="player.toggle()">
        <Pause v-if="player.isPlaying.value" class="recording__icon" />
        <Play v-else class="recording__icon" />
      </button>
      <time class="recording__time">{{ clock(player.position.value) }}</time>
      <Slider
        class="recording__slider"
        :model-value="player.position.value"
        :max="Math.max(player.duration.value, 1)"
        :step="100"
        @update:model-value="player.seek"
      />
      <time class="recording__time">{{ clock(player.duration.value) }}</time>
      <button
        type="button"
        class="recording__action"
        :data-on="isFollowing ? '' : undefined"
        aria-label="Follow"
        @click="isFollowing = !isFollowing"
      >
        <LocateFixed class="recording__icon" />
      </button>
    </div>
    <TranscriptView
      :cues="LECTURE_CUES"
      :active="active"
      :is-following="isFollowing"
      @seek="player.seek"
    />
  </div>
</template>

<style scoped>
.recording {
  display: flex;
  flex-direction: column;
  block-size: 100%;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

.recording__bar {
  display: flex;
  align-items: center;
  flex: none;
  gap: 1rem;
  min-block-size: var(--numen-action-size);
  padding-inline: var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.recording__slider {
  flex: 1;
}

.recording__time {
  color: var(--numen-hushed);
  font-variant-numeric: tabular-nums;
}

.recording__action {
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.recording__action[data-on],
.recording__action:first-child {
  color: var(--numen-ink);
}

.recording__action[data-on] {
  color: var(--numen-accent);
}

.recording__icon {
  inline-size: 1rem;
  block-size: 1rem;
  stroke-width: 1.875;
}
</style>
