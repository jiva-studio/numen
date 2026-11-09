<script setup lang="ts">
/**
 * A recording and its transcript: the player at the top of the pane, and the transcript text below it.
 *
 * The player is provided through the slot, as is any tab-specific content shown over it.
 */
import { computed } from 'vue'
import MediaActions from '@plugins/views-media-viewer/frontend/MediaActions.vue'
import Transcript from '@plugins/views-media-viewer/frontend/Transcript.vue'
import type { TranscriptState } from '@plugins/views-media-viewer/frontend/model/transcript'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    state: TranscriptState
    isFramed?: boolean
    isPlaying?: boolean | undefined
  }>(),
  { isFramed: false, isPlaying: undefined },
)

/* --------------------------------- State ---------------------------------- */
const { following } = props.state.content
const { timed } = props.state.lines
const { isPlaying: playbackIsPlaying } = props.state.playback

const isFollowing = computed(() => following.value)
const isPlaying = computed(() => props.isPlaying ?? playbackIsPlaying.value)

/* -------------------------------- Handlers -------------------------------- */
function onToggleFollow() {
  props.state.content.setFollowing(!isFollowing.value)
}
</script>

<template>
  <div class="media">
    <div class="media__head" :data-framed="props.isFramed ? '' : undefined">
      <slot name="player" />

      <MediaActions
        v-if="!props.isFramed && timed"
        :is-timed="timed"
        :is-following="isFollowing"
        @toggle-follow="onToggleFollow"
      />
    </div>

    <Transcript :state="props.state" :is-playing="isPlaying" />
  </div>
</template>

<style scoped>
.media {
  --media-apart: 1rem;
  --media-close: 0.25rem;
  container-type: size;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
}

/* The player heads the pane at its full width, on the rule that separates it from the words. */
.media__head {
  display: flex;
  align-items: center;
  flex: none;
  gap: var(--media-apart);
  inline-size: 100%;
  min-block-size: var(--numen-action-size);
  padding-inline: var(--numen-gutter);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

/* A framed media is as wide as the pane and stands on nothing: it is the strip. */
.media__head[data-framed] {
  display: block;
  flex: 0 1 auto;
  min-block-size: 0;
  overflow: hidden;
  padding-inline: 0;
  border-block-end: 0;
}
</style>
