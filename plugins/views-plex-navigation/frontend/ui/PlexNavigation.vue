<script setup lang="ts">
/** The Back and Forward buttons of a board; a button without a node to go to is disabled. */
import { createNamedIcon } from '@numen/ui'
import { WORDS } from '../words'

defineProps<{ canBack: boolean; canForward: boolean }>()

const emit = defineEmits<{
  (event: 'back'): void
  (event: 'forward'): void
}>()

const BackIcon = createNamedIcon('arrow-left')
const ForwardIcon = createNamedIcon('arrow-right')
</script>

<template>
  <div class="plex-navigation" role="group" aria-label="Navigation">
    <button
      type="button"
      class="plex-navigation__button"
      :aria-label="WORDS.back"
      :title="WORDS.back"
      :disabled="!canBack"
      @click="emit('back')"
    >
      <BackIcon class="plex-navigation__icon" />
    </button>
    <button
      type="button"
      class="plex-navigation__button"
      :aria-label="WORDS.forward"
      :title="WORDS.forward"
      :disabled="!canForward"
      @click="emit('forward')"
    >
      <ForwardIcon class="plex-navigation__icon" />
    </button>
  </div>
</template>

<style scoped>
.plex-navigation {
  display: inline-flex;
  gap: 0.25rem;
}

.plex-navigation__button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem;
  border: 0;
  border-radius: 0.375rem;
  background: transparent;
  color: var(--numen-hushed);
  cursor: pointer;
}

.plex-navigation__button:hover:not(:disabled) {
  color: var(--numen-ink);
}

.plex-navigation__button:disabled {
  opacity: 0.4;
  cursor: default;
}

.plex-navigation__icon {
  width: 1rem;
  height: 1rem;
}
</style>
