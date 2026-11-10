<script setup lang="ts">
/**
 * The picture of a video: the window's one media element, drawn here while this tab holds the
 * player, and given back to the player when it does not.
 */
import { onBeforeUnmount, useTemplateRef, watch } from 'vue'
import type { Player } from './model/player'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  readonly player: Player
  readonly isShown: boolean
}>()

/* --------------------------------- State ---------------------------------- */
const surface = useTemplateRef<HTMLElement>('surface')

/* --------------------------------- Hooks ---------------------------------- */
watch(
  [() => props.isShown, surface],
  ([isShown, container]) => {
    if (!container) return
    if (isShown) props.player.surface.attach(container)
    else props.player.surface.release(container)
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => {
  if (surface.value) props.player.surface.release(surface.value)
})
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
