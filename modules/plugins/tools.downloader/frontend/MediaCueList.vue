<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
export interface CueItem {
  readonly startMs: number
  readonly endMs: number
  readonly text: string
}

const props = defineProps<{
  readonly cues: readonly CueItem[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  seek: [ms: number]
}>()

/* -------------------------------- Helpers --------------------------------- */
function formatTime(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000)
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}
</script>

<template>
  <ul class="media-downloader__cue-list">
    <li
      v-for="(cue, index) in props.cues"
      :key="index"
      class="media-downloader__cue"
      @click="emit('seek', cue.startMs)"
    >
      <span class="media-downloader__cue-time">{{ formatTime(cue.startMs) }}</span>
      <span class="media-downloader__cue-text">{{ cue.text }}</span>
    </li>
  </ul>
</template>

<style scoped>
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
