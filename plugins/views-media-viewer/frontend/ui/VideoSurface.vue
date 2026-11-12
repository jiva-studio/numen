<script setup lang="ts">
/**
 * The picture of a video: the window's one media element, drawn here while this tab holds the
 * player, and given back to the player when it does not.
 */
import { useTemplateRef } from 'vue'
import type { Player } from '../model/player'
import { useVideoSurface } from '../model/useVideoSurface'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly player: Player
  readonly isShown: boolean
}>()

/* --------------------------------- State ---------------------------------- */
const surface = useTemplateRef<HTMLElement>('surface')

/* --------------------------------- Hooks ---------------------------------- */
useVideoSurface(
  surface,
  () => props.isShown,
  () => props.player.surface,
)
</script>

<template>
  <div ref="surface" class="recording__video" />
</template>

<style scoped>
.recording__video {
  display: flex;
  justify-content: center;
  flex: none;
}

.recording__video :deep(video) {
  --video-max-block-size: 24rem;
  inline-size: 100%;
  max-block-size: var(--video-max-block-size);
}
</style>
