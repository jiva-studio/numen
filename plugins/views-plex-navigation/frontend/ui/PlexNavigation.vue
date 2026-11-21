<script setup lang="ts">
/** The Back and Forward buttons of a board; a button without a node to go to is disabled. */
import { Tooltip, useTooltip } from '@numen/ui'
import { BackIcon, ForwardIcon } from '../model/icons'
import { WORDS } from '../words'

defineProps<{ canBack: boolean; canForward: boolean }>()

const emit = defineEmits<{
  (event: 'back'): void
  (event: 'forward'): void
}>()

const backTip = useTooltip()
const forwardTip = useTooltip()
</script>

<template>
  <div class="plex-navigation" role="group" aria-label="Navigation">
    <button
      type="button"
      class="plex-navigation__button"
      :aria-label="WORDS.back"
      :aria-describedby="backTip.describedBy.value"
      :disabled="!canBack"
      @mouseenter="backTip.open"
      @mouseleave="backTip.close"
      @click="emit('back')"
    >
      <BackIcon class="plex-navigation__icon" />
    </button>
    <button
      type="button"
      class="plex-navigation__button"
      :aria-label="WORDS.forward"
      :aria-describedby="forwardTip.describedBy.value"
      :disabled="!canForward"
      @mouseenter="forwardTip.open"
      @mouseleave="forwardTip.close"
      @click="emit('forward')"
    >
      <ForwardIcon class="plex-navigation__icon" />
    </button>
    <Teleport to="body">
      <Tooltip v-if="backTip.at.value" :id="backTip.id" :at="backTip.at.value">
        {{ WORDS.back }}
      </Tooltip>
      <Tooltip v-if="forwardTip.at.value" :id="forwardTip.id" :at="forwardTip.at.value">
        {{ WORDS.forward }}
      </Tooltip>
    </Teleport>
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
