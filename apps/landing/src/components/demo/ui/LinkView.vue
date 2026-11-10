<script setup lang="ts">
/**
 * A video at an address: the page at the top of the tab in a frame, and under it the transcript of what is
 * said in it.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { YOUTUBE_CUES, YOUTUBE_EMBED_URL, getActiveCueIndex } from '../lib/cues'
import MediaEmbed from './MediaEmbed.vue'
import TranscriptView from './TranscriptView.vue'

/* --------------------------------- State ---------------------------------- */
const WORDS = { playing: 'Playing media', taller: 'Resize player' }
const embed = useTemplateRef<{ seek(ms: number): void }>('embed')
const position = ref(0)
const active = computed(() => getActiveCueIndex(YOUTUBE_CUES, position.value))

/* -------------------------------- Handlers -------------------------------- */
function onSeek(ms: number) {
  position.value = ms
  embed.value?.seek(ms)
}
</script>

<template>
  <div class="link">
    <MediaEmbed
      ref="embed"
      class="link__embed"
      :embed="YOUTUBE_EMBED_URL"
      copy=""
      :words="WORDS"
      @time-update="position = $event"
    />
    <TranscriptView :cues="YOUTUBE_CUES" :active="active" is-following @seek="onSeek" />
  </div>
</template>

<style scoped>
.link {
  container-type: size;
  display: flex;
  flex-direction: column;
  block-size: 100%;
  min-block-size: 0;
  font-family: var(--numen-font-sans);
  color: var(--numen-ink);
}

.link__embed {
  flex: 0 1 auto;
  min-block-size: 0;
  overflow: hidden;
}
</style>
